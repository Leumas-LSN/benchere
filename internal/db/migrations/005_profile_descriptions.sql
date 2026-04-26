ALTER TABLE benchmark_profiles ADD COLUMN description TEXT NOT NULL DEFAULT '';
ALTER TABLE benchmark_profiles ADD COLUMN thresholds_json TEXT NOT NULL DEFAULT '';
ALTER TABLE benchmark_profiles ADD COLUMN is_builtin INTEGER NOT NULL DEFAULT 0;

UPDATE benchmark_profiles SET is_builtin = 1, description = 'Ecriture aleatoire 4K - Workload OLTP intensif, simule les ecritures de bases de donnees transactionnelles.' WHERE name = '4k_0read_100random';
UPDATE benchmark_profiles SET is_builtin = 1, description = 'Mixte 70% lecture / 30% ecriture aleatoire 4K - Typique des applications transactionnelles en production.' WHERE name = '4k_70read_100random';
UPDATE benchmark_profiles SET is_builtin = 1, description = 'Lecture aleatoire 4K - Simule les acces aleatoires en lecture : demarrages de VMs, caches froids, OLTP read-heavy.' WHERE name = '4k_100read_100random';
UPDATE benchmark_profiles SET is_builtin = 1, description = 'Mixte 50/50 aleatoire 8K - Workload equilibre pour bases de donnees mixtes (PostgreSQL, MySQL).' WHERE name = '8k_50read_100random';
UPDATE benchmark_profiles SET is_builtin = 1, description = 'Ecriture sequentielle 256K - Throughput maximal en ecriture : backups, logs, streaming video.' WHERE name = '256k_0read_0random';
UPDATE benchmark_profiles SET is_builtin = 1, description = 'Lecture sequentielle 256K - Throughput maximal en lecture : analytics, exports, replication.' WHERE name = '256k_100read_0random';
