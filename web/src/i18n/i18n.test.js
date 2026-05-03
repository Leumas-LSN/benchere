import { describe, it, expect } from 'vitest'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { parse } from '@babel/parser'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const FR_PATH = path.join(__dirname, 'fr.js')
const EN_PATH = path.join(__dirname, 'en.js')

// Walks an ObjectExpression AST node and returns:
// - duplicates: array of { path, firstLine, secondLine } when a key appears
//   twice at the SAME nesting level
// - paths: set of dotted leaf paths (any non-object value), used for parity
function walk(node, prefix = '', acc = { duplicates: [], paths: new Set() }) {
  if (!node || node.type !== 'ObjectExpression') return acc
  const seen = new Map()
  for (const prop of node.properties || []) {
    if (prop.type !== 'ObjectProperty' && prop.type !== 'Property') continue
    let key
    if (prop.key.type === 'Identifier') key = prop.key.name
    else if (prop.key.type === 'StringLiteral') key = String(prop.key.value)
    else continue
    const fullPath = prefix ? prefix + '.' + key : key
    if (seen.has(key)) {
      acc.duplicates.push({
        path: fullPath,
        firstLine: seen.get(key),
        secondLine: prop.key.loc.start.line,
      })
    } else {
      seen.set(key, prop.key.loc.start.line)
    }
    if (prop.value && prop.value.type === 'ObjectExpression') {
      walk(prop.value, fullPath, acc)
    } else {
      acc.paths.add(fullPath)
    }
  }
  return acc
}

function analyze(filePath) {
  const src = fs.readFileSync(filePath, 'utf8')
  const ast = parse(src, { sourceType: 'module' })
  for (const node of ast.program.body) {
    if (node.type === 'ExportDefaultDeclaration' && node.declaration && node.declaration.type === 'ObjectExpression') {
      return walk(node.declaration)
    }
  }
  return { duplicates: [], paths: new Set() }
}

describe('i18n locales', () => {
  const fr = analyze(FR_PATH)
  const en = analyze(EN_PATH)

  it('fr.js has no duplicate keys at any nesting level', () => {
    if (fr.duplicates.length > 0) {
      const lines = fr.duplicates.map(d => `${d.path} (lines ${d.firstLine} and ${d.secondLine})`)
      throw new Error('Duplicate keys found in fr.js:\n  ' + lines.join('\n  '))
    }
    expect(fr.duplicates).toHaveLength(0)
  })

  it('en.js has no duplicate keys at any nesting level', () => {
    if (en.duplicates.length > 0) {
      const lines = en.duplicates.map(d => `${d.path} (lines ${d.firstLine} and ${d.secondLine})`)
      throw new Error('Duplicate keys found in en.js:\n  ' + lines.join('\n  '))
    }
    expect(en.duplicates).toHaveLength(0)
  })

  it('fr and en have identical key trees', () => {
    const onlyFr = [...fr.paths].filter(k => !en.paths.has(k)).sort()
    const onlyEn = [...en.paths].filter(k => !fr.paths.has(k)).sort()
    if (onlyFr.length || onlyEn.length) {
      throw new Error(
        'FR/EN key parity violated.\n' +
        '  Only in FR (' + onlyFr.length + '): ' + onlyFr.join(', ') + '\n' +
        '  Only in EN (' + onlyEn.length + '): ' + onlyEn.join(', ')
      )
    }
    expect(onlyFr).toHaveLength(0)
    expect(onlyEn).toHaveLength(0)
    expect(fr.paths.size).toBe(en.paths.size)
  })
})
