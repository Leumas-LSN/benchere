---
name: Benchere
description: Bench d'opération pour ingénieurs en virtualisation. La donnée passe avant le décor.
colors:
  ember-orange:    "#f97316"
  ember-deep:      "#c2410c"
  ember-glow:      "#fff7ed"
  ember-glow-dark: "#1a0d05"
  graphite-50:     "#fafafa"
  graphite-100:    "#f5f5f5"
  graphite-200:    "#e5e5e5"
  graphite-300:    "#d4d4d4"
  graphite-400:    "#a3a3a3"
  graphite-500:    "#737373"
  graphite-600:    "#525252"
  graphite-700:    "#404040"
  graphite-800:    "#262626"
  graphite-900:    "#171717"
  graphite-950:    "#0a0a0a"
  surface-dark:    "#111113"
  surface-elev-dark: "#18181b"
  border-dark:     "#26262a"
  signal-pass:     "#16a34a"
  signal-pass-dark:"#4ade80"
  signal-fail:     "#dc2626"
  signal-fail-dark:"#f87171"
  signal-active:   "#2563eb"
  signal-active-dark:"#60a5fa"
  signal-warn:     "#d97706"
  signal-warn-dark:"#fbbf24"
  data-cool:       "#0ea5e9"
  data-violet:     "#7c3aed"
typography:
  display:
    fontFamily:    "Geist, ui-sans-serif, system-ui, sans-serif"
    fontSize:      "1.5rem"
    fontWeight:    600
    lineHeight:    "1.25"
    letterSpacing: "-0.01em"
  headline:
    fontFamily:    "Geist, ui-sans-serif, system-ui, sans-serif"
    fontSize:      "1rem"
    fontWeight:    600
    lineHeight:    "1.4"
  title:
    fontFamily:    "Geist, ui-sans-serif, system-ui, sans-serif"
    fontSize:      "0.875rem"
    fontWeight:    600
    lineHeight:    "1.4"
  body:
    fontFamily:    "Geist, ui-sans-serif, system-ui, sans-serif"
    fontSize:      "0.875rem"
    fontWeight:    400
    lineHeight:    "1.5"
  label:
    fontFamily:    "Geist, ui-sans-serif, system-ui, sans-serif"
    fontSize:      "0.6875rem"
    fontWeight:    600
    lineHeight:    "1.4"
    letterSpacing: "0.08em"
  mono:
    fontFamily:    "\"Geist Mono\", ui-monospace, SFMono-Regular, Menlo, monospace"
    fontSize:      "0.875rem"
    fontWeight:    500
    lineHeight:    "1.4"
    fontFeature:   "\"tnum\""
rounded:
  sm:   "4px"
  md:   "6px"
  lg:   "8px"
  xl:   "12px"
  2xl:  "16px"
  pill: "9999px"
spacing:
  xs:  "4px"
  sm:  "8px"
  md:  "12px"
  lg:  "16px"
  xl:  "20px"
  2xl: "24px"
  3xl: "32px"
  4xl: "40px"
components:
  button-primary:
    backgroundColor: "{colors.ember-orange}"
    textColor:       "#ffffff"
    typography:      "{typography.title}"
    rounded:         "{rounded.lg}"
    padding:         "0 14px"
    height:          "36px"
  button-primary-hover:
    backgroundColor: "{colors.ember-deep}"
  button-primary-active:
    backgroundColor: "#9a3412"
  button-secondary:
    backgroundColor: "#ffffff"
    textColor:       "{colors.graphite-900}"
    typography:      "{typography.title}"
    rounded:         "{rounded.lg}"
    padding:         "0 14px"
    height:          "36px"
  button-ghost:
    backgroundColor: "transparent"
    textColor:       "{colors.graphite-600}"
    typography:      "{typography.title}"
    rounded:         "{rounded.lg}"
    padding:         "0 14px"
    height:          "36px"
  button-danger:
    backgroundColor: "#dc2626"
    textColor:       "#ffffff"
    typography:      "{typography.title}"
    rounded:         "{rounded.lg}"
    padding:         "0 14px"
    height:          "36px"
  card:
    backgroundColor: "#ffffff"
    rounded:         "{rounded.xl}"
    padding:         "20px"
  card-flush:
    backgroundColor: "#ffffff"
    rounded:         "{rounded.xl}"
    padding:         "0"
  input:
    backgroundColor: "#ffffff"
    textColor:       "{colors.graphite-900}"
    typography:      "{typography.body}"
    rounded:         "{rounded.lg}"
    padding:         "0 12px"
    height:          "40px"
  pill:
    backgroundColor: "{colors.graphite-100}"
    textColor:       "{colors.graphite-600}"
    typography:      "{typography.label}"
    rounded:         "{rounded.md}"
    padding:         "2px 8px"
  nav-link:
    backgroundColor: "transparent"
    textColor:       "{colors.graphite-600}"
    typography:      "{typography.title}"
    rounded:         "{rounded.lg}"
    padding:         "0 12px"
    height:          "40px"
  nav-link-active:
    backgroundColor: "{colors.ember-glow}"
    textColor:       "{colors.graphite-900}"
  status-pill-pass:
    backgroundColor: "#f0fdf4"
    textColor:       "{colors.signal-pass}"
    typography:      "{typography.label}"
    rounded:         "{rounded.md}"
    padding:         "2px 8px"
  status-pill-fail:
    backgroundColor: "#fef2f2"
    textColor:       "{colors.signal-fail}"
    typography:      "{typography.label}"
    rounded:         "{rounded.md}"
    padding:         "2px 8px"
---

# Design System: Benchere

## 1. Overview

**Creative North Star: "The Operations Bench."**

Benchere se présente comme un poste d'opération : un banc d'instruments alignés sous la main d'un ingénieur. Aucun spectacle, aucune mise en scène. La hiérarchie visuelle existe pour qu'on lise une valeur, qu'on prenne une décision, et qu'on passe au job suivant. L'interface s'efface derrière la donnée. L'orange n'apparaît que là où l'œil doit se poser : action principale, route active, marqueur dans la marge, accent du rapport. Partout ailleurs, des neutres tièdement tintés (légèrement chauds) qui posent le décor sans le revendiquer.

Le système rejette frontalement quatre familles esthétiques nommées dans `PRODUCT.md` : les gradients pastel des landings AI, le néon des dashboards crypto, le bleu corporate fade des intranets 2014, et la grammaire marketing SaaS (hero numbers décoratifs, testimonials, three-column features). Aucune de ces silhouettes ne doit pouvoir être confondue avec Benchere au premier coup d'œil. La référence positive est double : densité d'un Datadog, soin d'un Linear. Pas de compromis, les deux qualités cohabitent.

**Key Characteristics:**

- Density-first : tableaux compacts, alignement à droite des chiffres, peu de whitespace gratuit dans les rangées de données.
- Bordures plutôt qu'ombres pour la séparation des surfaces de page (élévation hybride, voir §4).
- Une seule couleur de marque, utilisée comme signature, jamais comme décoration.
- Vocabulaire neutre légèrement chaud (chroma ~0.005 vers l'orange) plutôt que zinc froid.
- Toute animation a une raison fonctionnelle. Les pulses informent ; rien ne bouge pour faire joli.

## 2. Colors: The Ember & Graphite Palette

Trois familles, trois rôles. La famille Ember pour la marque (rare). Graphite pour tout le décor (omniprésent). Signal pour les statuts (sémantique uniquement).

### Primary

- **Ember Orange** (`#f97316`) : marque. CTA primaire, route active de la sidebar, marqueur dans le gutter, indicateur de progression, traits dans le rapport PDF (titres section, charts, stripe de couverture).
- **Ember Deep** (`#c2410c`) : Ember Orange en hover/active sur les boutons primaires, et accent lourd dans le rapport PDF (logo carré, titres de meta).
- **Ember Glow** (`#fff7ed` clair / `#1a0d05` sombre) : surface tintée d'orange à très faible saturation. Background du nav-link actif et de la zone "configurer Proxmox" pour signaler doucement la zone tactique sans crier.

### Neutral (Graphite)

Échelle de gris légèrement tiède (chroma minuscule poussée vers l'orange) pour rester cohérente avec la marque sans entrer dans le territoire "warm beige".

- **Graphite 50** (`#fafafa`) : canvas en mode clair (le fond de la page).
- **Graphite 100** (`#f5f5f5`) : surface "muted" — entête de tableau, fond de pill, fond de bouton secondaire au hover.
- **Graphite 200** (`#e5e5e5`) : bordure par défaut. Encadre les cartes, sépare les sections, dessine les inputs.
- **Graphite 300** (`#d4d4d4`) : bordure renforcée — hover sur cartes/boutons secondaires.
- **Graphite 400 / 500** (`#a3a3a3` / `#737373`) : texte estompé, légendes, valeurs neutres, métadonnées de timestamp.
- **Graphite 600** (`#525252`) : texte secondaire (sous-titres, descriptions de cartes, helpers).
- **Graphite 900** (`#171717`) : texte primaire en mode clair. Tout ce qu'on lit en premier.
- **Graphite 950** (`#0a0a0a`) : canvas en mode sombre.
- **Surface Dark** (`#111113`) : surface principale en mode sombre (sidebar, topbar, cartes). Légèrement remontée pour démarquer du canvas.
- **Surface Elev Dark** (`#18181b`) : élévation supérieure en mode sombre (modales, popovers).
- **Border Dark** (`#26262a`) : bordure par défaut en mode sombre.

### Signal (status uniquement)

Couleurs sémantiques. Jamais utilisées en décoration.

- **Signal Pass** (`#16a34a` clair / `#4ade80` sombre) : statut `done`, verdict `pass`, succès.
- **Signal Fail** (`#dc2626` clair / `#f87171` sombre) : statut `failed`, verdict `fail`, action destructrice.
- **Signal Active** (`#2563eb` clair / `#60a5fa` sombre) : statut `running`, alerte info.
- **Signal Warn** (`#d97706` clair / `#fbbf24` sombre) : statut `provisioning`, alerte avertissement, configuration manquante.

### Data viz

- **Data Cool** (`#0ea5e9`) : seconde série dans les charts (RAM, CPU cluster).
- **Data Violet** (`#7c3aed`) : troisième série (latence). Distincte de l'Ember pour rester lisible côte à côte.

### Named Rules

**The One Voice Rule.** L'Ember Orange ne couvre jamais plus de ~10% de la surface visible. Si tu peins plus, tu casses la signature. Si tout est orange, plus rien ne l'est.

**The Tinted Neutral Rule.** Aucun gris n'est purement neutre. La famille Graphite est légèrement poussée vers l'orange (chroma ~0.005), pour que blanc et noir soient cousins et non étrangers à la marque. Les valeurs `#000` et `#fff` purs sont prohibées.

**The Signal-Stays-Sematic Rule.** Les couleurs Signal ne servent qu'à transporter du sens (statut, verdict, alerte). Elles ne décorent jamais une carte par "joliesse", n'illustrent pas une icône hors contexte de statut, et ne se mélangent pas dans les charts comme s'il s'agissait de séries arbitraires.

## 3. Typography

**Display Font:** Geist (avec `ui-sans-serif, system-ui, -apple-system, "Segoe UI", sans-serif` en fallback)
**Body Font:** Geist (même famille, même graisse 400)
**Mono Font:** Geist Mono (avec `ui-monospace, SFMono-Regular, Menlo, monospace`)

**Character.** Geist a été choisi pour son équilibre entre rigueur géométrique et chaleur résiduelle. Pas Inter (trop générique, trop "AI startup"), pas Roboto (trop Material), pas IBM Plex (trop "lab"). Geist Mono complète : largeurs fixes pour aligner les colonnes de chiffres, glyphes 0/O/1/l distincts, lecture fluide même à 11px.

### Hierarchy

- **Display** (Geist 600 — 1.5rem / 24px — line-height 1.25 — letter-spacing -0.01em) : `h1` du `PageHeader`, en haut de chaque vue. Une seule occurrence par page.
- **Headline** (Geist 600 — 1rem / 16px — line-height 1.4) : titre de section au sein d'une carte ("Identification", "Workers", "Mode de benchmark"). Posé, pas crié.
- **Title** (Geist 600 — 0.875rem / 14px — line-height 1.4) : intitulés de boutons, items de navigation, lignes en gras dans les listes.
- **Body** (Geist 400 — 0.875rem / 14px — line-height 1.5 — max 65ch) : texte courant, descriptions, helpers. Toute prose suit cette graisse.
- **Label** (Geist 600 — 0.6875rem / 11px — letter-spacing 0.08em — UPPERCASE) : eyebrows ("Vue d'ensemble", "Étape 2 / 4"), titres de cartes ("RUNS EN COURS"), pills de statut. Rare et précieux.
- **Mono** (Geist Mono 500 — 0.875rem / 14px — `font-feature: "tnum"`) : tous les chiffres mesurés (IOPS, latence, débit, %, durées), tous les identifiants (job ID, IP, VM ID), toutes les commandes inline et noms de profils techniques.

### Named Rules

**The Number-In-Mono Rule.** Tout chiffre qui sera comparé verticalement (colonnes de tableaux, KPI cards alignés en grille, métriques live) est rendu en Geist Mono avec `tabular-nums`. Le chiffre seul prouve qu'il est lu, pas l'environnement qui l'entoure.

**The Single-Display Rule.** Un seul `Display` par page (le titre du `PageHeader`). Ne pas multiplier les gros titres : si une section veut peser plus, elle prend `Headline`, jamais `Display`.

**The Label-Earns-Caps Rule.** L'UPPERCASE est réservé au token `Label` (eyebrows, card titles, pills). Aucun autre niveau ne porte de majuscules. Le bouton ne crie pas. La donnée ne crie pas.

## 4. Elevation

Le système est **hybride et déclaratif** : les surfaces de page sont à plat, séparées par des bordures fines plutôt que par des ombres. Les éléments **décollés du flux** (modales, popovers, dropdowns potentiels) prennent une ombre marquée, parce qu'ils sont littéralement au-dessus du contenu.

Une carte n'a pas besoin de flotter pour être une carte : la bordure de 1px et le léger décalage de fond (`#ffffff` sur canvas `#fafafa`) suffisent à la délimiter. Cette discipline garde la lecture stable et évite l'effet "dashboard rempli de cartes en lévitation" typique des templates SaaS génériques.

### Shadow Vocabulary

- **Shadow Card** (`box-shadow: 0 1px 3px 0 rgb(0 0 0 / 0.06), 0 1px 2px -1px rgb(0 0 0 / 0.06)`) : posé par défaut sur toutes les `card` et `surface`. Quasi imperceptible, sert juste à éviter que la carte se confonde au pixel près avec le canvas en mode clair. En mode sombre, l'ombre devient une variante presque transparente avec un highlight intérieur (`0 1px 0 0 rgb(255 255 255 / 0.02)`).
- **Shadow Pop** (`box-shadow: 0 10px 30px -12px rgb(0 0 0 / 0.18), 0 4px 10px -4px rgb(0 0 0 / 0.10)`) : modales, panneaux d'overlay. Net décalage de hauteur, le contenu derrière est manifestement secondaire.
- **Shadow Brand** (`box-shadow: 0 6px 18px -6px rgb(249 115 22 / 0.55)`) : exclusif au CTA primaire au hover. Halo orange diffus qui matérialise l'engagement de l'action.

### Named Rules

**The Border-Before-Shadow Rule.** La hiérarchie de surface se gagne d'abord par bordure et fond, ensuite seulement par ombre. Une carte qui ajoute une ombre marquée pour exister est probablement mal posée dans la grille.

**The Lifted-Means-Detached Rule.** Une ombre marquée signifie que l'élément est physiquement détaché du flux (modale, dropdown). Si le contenu reste dans le flux normal de la page, il ne mérite pas une `Shadow Pop`.

## 5. Components

Caractère général : **operational and confident.** Géométrie franche, états marqués, transitions courtes (150–220ms, ease-out exponentiel), pas d'enrobage décoratif. Les composants se présentent comme des contrôles d'instrument, pas comme des objets.

### Buttons

- **Shape** : coins arrondis modérés (8px / `rounded-lg`). Aucune surface en `pill` (sauf badges de statut). Hauteur fixe par taille (`h-9` / 36px par défaut, `h-8` / 32px pour `btn-sm`, `h-11` / 44px pour `btn-lg`).
- **Primary** : `Ember Orange` (#f97316) sur fond, texte blanc, `Title` typography. Hover passe à `Ember Deep` (#c2410c) avec halo `Shadow Brand`. Active descend à `#9a3412`. Une seule action primaire par vue.
- **Secondary** : surface (#ffffff clair / #18181b sombre), texte primaire (`Graphite 900` / `Graphite 100`), bordure `Graphite 200` / `Border Dark`. Hover : bordure `Graphite 300`, fond `Graphite 100`. Action neutre, présente partout.
- **Ghost** : fond transparent, texte `Graphite 600`. Hover : fond `Graphite 100`, texte primaire. Action discrète, pour les liens d'action dans les tableaux et les barres d'outils.
- **Danger** : rouge `#dc2626` plein, texte blanc. Réservé aux actions destructrices avérées (Stop d'un job en cours). La variante "Danger Ghost" (texte rouge sur fond transparent, hover fond rouge à 5% d'opacité) sert pour les actions destructrices secondaires comme "Vider l'historique".
- **Focus** : anneau `outline: 2px solid Ember Orange` avec `outline-offset: 2px`. Visible sur tous les boutons, jamais retiré globalement.

### Status Pills (signature component)

Combinaison icône-dot + label, partout où un statut sémantique est affiché. La couleur seule ne porte pas le sens.

- **Style** : badge inline (`px-2 py-0.5`, `rounded-md`), `Label` typography, ring intérieur 1px à 20% d'opacité de la couleur sémantique. Le dot pulse (`animate-pulse-dot`, 1.6s) lorsque le statut est actif (`running`, `provisioning`, `pending`).
- **Variantes** : Pass (vert), Fail (rouge), Running (bleu), Provisioning (ambre), Pending (gris), Cancelled (gris). Chacune respecte le contraste 4.5:1 du label sur fond.
- **Position** : à côté du nom du job (jamais centré), aligné à la baseline du texte parent.

### Cards / Surfaces

- **Corner Style** : 12px (`rounded-xl`). Plus arrondi que les boutons mais sans virer "soft".
- **Background** : surface (`#ffffff` clair / `#111113` sombre).
- **Shadow Strategy** : `Shadow Card` par défaut (voir §4). Pas d'élévation au hover sur les cartes de contenu.
- **Border** : 1px `Graphite 200` / `Border Dark`. Toujours présente, c'est la stratégie d'élévation principale.
- **Internal Padding** : 20px (`p-5`). Variante `card-flush` pour cartes avec entête séparé et tableau plein largeur (padding 0 sur le wrapper, padding interne géré par `card-header`).
- **Nesting interdit** : aucune carte dans une carte. Si une zone d'une carte mérite sa propre identité visuelle, utiliser `surface-muted` (fond `Graphite 100` / `Surface Elev Dark`, bordure `Graphite 200` / `Border Subtle`, sans ombre).

### Inputs

- **Style** : hauteur 40px (`h-10`), padding horizontal 12px, bordure 1px `Graphite 200`, fond surface, texte `Body`. Rayon 8px (cohérent avec les boutons).
- **Focus** : bordure passe à `Ember Orange`, halo `ring-2 ring-brand-500/40` (Ember à 40% d'opacité, 2px). Ni glow saturé ni shift de layout.
- **Placeholder** : `Graphite 400`. Suggestion uniquement, jamais en remplacement du label.
- **Helper** : `text-xs` `Graphite 500` sous l'input (`.helper`). Toujours visible, jamais conditionnel.
- **Disabled** : opacité 50%, `cursor: not-allowed`.
- **Le label vit au-dessus de l'input**, en `Label` typography (uppercase, 11px, tracking 0.08em). Aucun placeholder-only.

### Navigation

- **Sidebar** : largeur fixe (240px / 60px replié), fond surface, bordure droite 1px. Hauteur de l'app entière, scrollable indépendamment.
- **Nav link** : hauteur 40px, padding horizontal 12px, rayon 8px, `Title` typography, icône SVG 18px à gauche.
- **Default** : texte `Graphite 600`. Hover : fond `Graphite 100`, texte primaire.
- **Active** : fond `Ember Glow` (#fff7ed / #1a0d05), texte primaire, plus un marqueur orange de 3px de large dans le gutter à -8px (rendu via `::before`, indépendant du link). Le marqueur se lit comme une encoche sur le banc, pas comme une bordure colorée sur le bouton.
- **Topbar** : hauteur 64px, breadcrumb dynamique à gauche (icône + label + sous-segment), CTA primaire et toggle thème à droite.

### Tables

Composant central de Benchere. Densité haute, pas d'ornement.

- **Header** : `Label` typography (uppercase 11px tracking 0.08em), fond `Graphite 100` / surface mutée en sombre, bordure inférieure 1px.
- **Cells** : 12px padding vertical, 16px horizontal. Bordure inférieure subtle (`Graphite 200` à 50%) entre rangées.
- **Row hover** : fond `Graphite 100` (clair) / `Surface Muted` (sombre). Pas de transformation, pas de shift.
- **Numerical columns** : alignées à droite, en `Mono`. Le chiffre porte la lecture verticale.
- **Action column** : alignée à droite, padding-right 20px, boutons Ghost groupés serrés.

### Charts

- **Background** : aucun. Le canvas du chart est transparent, posé sur la `card` parente.
- **Grid lines** : `Graphite 200` clair (à 50% en dark). 3 ticks max sur l'axe Y.
- **Line** : Ember Orange par défaut sur les charts d'IOPS. Un seul autre coloris par chart (Data Violet pour la latence, Data Cool pour la RAM/CPU cluster). Stroke 2px, joins arrondis.
- **Area fill** : gradient vertical 32% → 2% d'opacité de la couleur de la ligne. Donne du volume sans masquer la donnée.
- **Tooltip** : surface `Graphite 900` (clair) / `#1c1c1f` (sombre), texte `#fafafa`, bordure transparente, padding 10px, valeur en `Mono`.
- **No legend** : le titre de la card et la couleur de la ligne suffisent quand il n'y a qu'une série.

### Modal (sparingly)

- **Backdrop** : `rgba(15, 15, 18, 0.45)` clair / `rgba(0, 0, 0, 0.65)` sombre, blur 4px (rare exception au "no glassmorphism" : strictement pour signifier la suspension du contexte derrière, jamais comme effet décoratif).
- **Panel** : `Surface Elev` (`#ffffff` / `#18181b`), bordure 1px `Border Default`, rayon 16px (`rounded-2xl`), `Shadow Pop`, max-width 32rem.
- **Animation** : fade-in 220ms ease-out. Pas de scale, pas de bounce.
- **Usage** : import de profil, confirmations destructrices. Jamais pour la navigation principale.

## 6. Do's and Don'ts

### Do

- **Do** réserver l'Ember Orange à l'action primaire, à la route active de sidebar, au marqueur de progression, et aux ancres du rapport PDF. Le reste vit en Graphite.
- **Do** rendre tous les chiffres en `Geist Mono` avec `font-feature: "tnum"` et alignement à droite dans les tableaux. La lecture verticale en dépend.
- **Do** matérialiser le statut par dot + couleur + label. Trois canaux pour un sens, jamais un seul.
- **Do** privilégier les bordures fines et le décalage de fond par rapport aux ombres lourdes pour séparer les surfaces (`The Border-Before-Shadow Rule`).
- **Do** soigner le rapport PDF autant que l'app vivante : c'est l'artefact qui finit dans les mains du client, à la livraison.
- **Do** respecter `prefers-reduced-motion` : toutes les animations (pulse-dot, shimmer skeleton, fade-in modal) doivent se neutraliser quand l'utilisateur l'a demandé.
- **Do** afficher une seule action primaire par vue. Les autres actions sont Secondary, Ghost ou Danger.

### Don't

- **Don't** utiliser de gradient pastel violet/rose, ni de glassmorphism décoratif, ni de hero cards centrées sur fond flou. C'est l'esthétique "AI startup landing 2023" frontalement bannie par PRODUCT.md.
- **Don't** virer "cyber neon" : pas de fond noir saturé avec accents néon vert/bleu, pas de gros chiffres clignotants. Benchere n'est pas un terminal de trader.
- **Don't** glisser dans le "Bootstrap corporate" : pas de bleu administratif fade, pas de tableaux serrés sans hiérarchie, pas de badges Bootstrap par défaut.
- **Don't** importer la grammaire marketing SaaS : pas de "hero metric" décoratif, pas de three-column feature cards identiques, pas de testimonial blocks. Ce n'est pas une landing.
- **Don't** utiliser de `border-left` ou `border-right` supérieur à 1px comme accent coloré sur les cartes, alertes ou items de liste. Le marqueur de la sidebar est l'unique exception (et il vit dans le gutter, pas sur l'élément).
- **Don't** appliquer `background-clip: text` sur un gradient pour faire du texte coloré. Aucun cas d'usage ne le justifie ici.
- **Don't** imbriquer une carte dans une carte. Si une zone mérite son propre traitement, utilise `surface-muted` (fond tinté sans ombre).
- **Don't** animer pour décorer. Une animation existe pour informer (statut actif qui pulse, progression qui se remplit, route qui se charge en fade). Sinon, immobile.
- **Don't** utiliser `#000` ou `#fff` purs. Tout neutre est tinté Graphite (chroma minuscule vers l'orange).
- **Don't** mettre plus d'une couleur de marque par vue, ni plus d'un Display par page (`The Single-Display Rule`).
- **Don't** traduire un statut par la couleur seule. Chaque pill porte un label texte ; le verdict pass/fail du rapport reste lisible imprimé en noir et blanc.
- **Don't** céder au tiret cadratin (—) dans la copy d'interface ni dans les commentaires de code. Préfère virgule, deux-points, point-virgule, parenthèse.
