INSERT OR IGNORE INTO benchmark_profiles (id, name, config_json) VALUES
  ('default-1', '4k_0read_100random', '{"block_size":"4k","read_pct":0,"random_pct":100}'),
  ('default-2', '4k_70read_100random', '{"block_size":"4k","read_pct":70,"random_pct":100}'),
  ('default-3', '4k_100read_100random', '{"block_size":"4k","read_pct":100,"random_pct":100}'),
  ('default-4', '8k_50read_100random', '{"block_size":"8k","read_pct":50,"random_pct":100}'),
  ('default-5', '256k_0read_0random', '{"block_size":"256k","read_pct":0,"random_pct":0}'),
  ('default-6', '256k_100read_0random', '{"block_size":"256k","read_pct":100,"random_pct":0}');
