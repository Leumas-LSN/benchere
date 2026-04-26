-- Replace JSON stub profiles with real elbencho configs
UPDATE benchmark_profiles SET config_json = '# Fichier d''exmple de profil pour un benchmark via l''outil elbencho.
# Pour une liste des commandes disponnibles utiliser la commande elbencho --help-all
# Profil: 4k_0read_100random = Ecriture intensive petits blocs (test extrême des 4 coins)

#BASE
#Number of I/O worker threads. (Default: 1)
threads=8
#Depth of I/O queue per thread for asynchronous I/O. Setting this to 2 or higher turns on async I/O. (Default: 1)
iodepth=4
#Time limit in seconds for each benchmark phase. If the limit is exceeded for a phase then no further phases will run. (Default: 0 for disabled)
timelimit=300
#Number of bytes to read/write in a single operation. Each thread needs to keep one block in RAM (or multiple blocks if "--iodepth" is used), so be careful with large block sizes. (Default: 1M; supports base2 suffixes, e.g. "128K")
block=4k
#se direct IO (also known as O_DIRECT) to avoid file contents caching. Note: For network or cluster filesystems, it depends on the actual filesystem whether this option is only effective for the client-side cache or also for the server-side cache. Also, some filesystems might ignore this completely or might have a RAID controller cache which operates independent of this setting.
direct=1
#Let I/O threads run in an infinite loop, i.e. they restart from the beginning when the reach the end of the specified workload. Terminate this via ctrl+c or by using "--timelimit"
infloop=1
#Preallocate file disk space in a write phase via posix_fallocate().
preallocfile=1
#Benchmark paths are not shared between service instances. Thus, each service instance will work on its own full dataset instead of a fraction of the data set.
nosvcshare=1


#AFFICHAGE
#Show minimum, average and maximum latency for read/write operations and entries. In read and write phases, entry latency includes file open, read/write and>
lat=1
#Show CPU utilization in phase stats results.
cpu=1
#Show latency histogram.
lathisto=0


#PROFILE
#Read files.
#read=1
write=1
#Read/write at random offsets.
rand=1
#Random number algorithm for "--rand". Values: "fast" for high speed but weaker randomness; "balanced_single" for good balance of speed and randomness; "strong" for high CPU cost but strong randomness. (Default: a special algo for maximum single pass block coverage in write phase for aligned IO and "balanced_single" for reads and unaligned IO)
randalgo=balanced_single
#Percentage of blocks that should be read in a write phase. (Default: 0; Max: 100)
rwmixpct=0
#Random number algorithm for "--blockvarpct". Values: "fast" for high speed but weaker randomness; "balanced" for good balance of speed and randomness; "str>
blockvaralgo=balanced
#Block variance percentage. Defines the percentage of each block that will be refilled with random data between writes. This can be used to defeat compressi>
blockvarpct=100' WHERE name = '4k_0read_100random';
UPDATE benchmark_profiles SET config_json = '# Fichier d''exmple de profil pour un benchmark via l''outil elbencho.
# Pour une liste des commandes disponnibles utiliser la commande elbencho --help-all
# Profil: 4k_70read_100random = Test d''émulation des charges de travail les plus répandues

#BASE
#Number of I/O worker threads. (Default: 1)
threads=8
#Depth of I/O queue per thread for asynchronous I/O. Setting this to 2 or higher turns on async I/O. (Default: 1)
iodepth=4
#Time limit in seconds for each benchmark phase. If the limit is exceeded for a phase then no further phases will run. (Default: 0 for disabled)
timelimit=300
#Number of bytes to read/write in a single operation. Each thread needs to keep one block in RAM (or multiple blocks if "--iodepth" is used), so be careful with large block sizes. (Default: 1M; supports base2 suffixes, e.g. "128K")
block=4k
#Use direct IO (also known as O_DIRECT) to avoid file contents caching. Note: For network or cluster filesystems, it depends on the actual filesystem whether this option is only effective for the client-side cache or also for the server-side cache. Also, some filesystems might ignore this completely or might have a RAID controller cache which operates independent of this setting.
direct=1
#Let I/O threads run in an infinite loop, i.e. they restart from the beginning when the reach the end of the specified workload. Terminate this via ctrl+c or by using "--timelimit"
infloop=1
#Preallocate file disk space in a write phase via posix_fallocate().
preallocfile=1
#Benchmark paths are not shared between service instances. Thus, each service instance will work on its own full dataset instead of a fraction of the data set.
nosvcshare=1


#AFFICHAGE
#Show minimum, average and maximum latency for read/write operations and entries. In read and write phases, entry latency includes file open, read/write and>
lat=1
#Show CPU utilization in phase stats results.
cpu=1
#Show latency histogram.
lathisto=0


#PROFILE
#Read files.
#read=1
#Write files. Create them if they don''t exist.
write=1
#Read/write at random offsets.
rand=1
#Random number algorithm for "--rand". Values: "fast" for high speed but weaker randomness; "balanced_single" for good balance of speed and randomness; "strong" for high CPU cost but strong randomness. (Default: a special algo for maximum single pass block coverage in write phase for aligned IO and "balanced_single" for reads and unaligned IO)
randalgo=balanced_single
#Percentage of blocks that should be read in a write phase. (Default: 0; Max: 100)
rwmixpct=70
#Random number algorithm for "--blockvarpct". Values: "fast" for high speed but weaker randomness; "balanced" for good balance of speed and randomness; "str>
blockvaralgo=balanced
#Block variance percentage. Defines the percentage of each block that will be refilled with random data between writes. This can be used to defeat compressi>
blockvarpct=100' WHERE name = '4k_70read_100random';
UPDATE benchmark_profiles SET config_json = '# Fichier d''exmple de profil pour un benchmark via l''outil elbencho.
# Pour une liste des commandes disponnibles utiliser la commande elbencho --help-all
# Profil: 4k_100read_100random = Lecture intensive petits blocs (test extrême des 4 coins)

#BASE
#Number of I/O worker threads. (Default: 1)
threads=8
#Depth of I/O queue per thread for asynchronous I/O. Setting this to 2 or higher turns on async I/O. (Default: 1)
iodepth=4
#Time limit in seconds for each benchmark phase. If the limit is exceeded for a phase then no further phases will run. (Default: 0 for disabled)
timelimit=300
#Number of bytes to read/write in a single operation. Each thread needs to keep one block in RAM (or multiple blocks if "--iodepth" is used), so be careful with large block sizes. (Default: 1M; supports base2 suffixes, e.g. "128K")
block=4k
#se direct IO (also known as O_DIRECT) to avoid file contents caching. Note: For network or cluster filesystems, it depends on the actual filesystem whether this option is only effective for the client-side cache or also for the server-side cache. Also, some filesystems might ignore this completely or might have a RAID controller cache which operates independent of this setting.
direct=1
#Let I/O threads run in an infinite loop, i.e. they restart from the beginning when the reach the end of the specified workload. Terminate this via ctrl+c or by using "--timelimit"
infloop=1
#Preallocate file disk space in a write phase via posix_fallocate().
preallocfile=1
#Benchmark paths are not shared between service instances. Thus, each service instance will work on its own full dataset instead of a fraction of the data set.
nosvcshare=1


#AFFICHAGE
#Show minimum, average and maximum latency for read/write operations and entries. In read and write phases, entry latency includes file open, read/write and>
lat=1
#Show CPU utilization in phase stats results.
cpu=1
#Show latency histogram.
lathisto=0


#PROFILE
#Read files.
read=1
#write=1
#Read/write at random offsets.
rand=1
#Random number algorithm for "--rand". Values: "fast" for high speed but weaker randomness; "balanced_single" for good balance of speed and randomness; "strong" for high CPU cost but strong randomness. (Default: a special algo for maximum single pass block coverage in write phase for aligned IO and "balanced_single" for reads and unaligned IO)
randalgo=balanced_single
#Percentage of blocks that should be read in a write phase. (Default: 0; Max: 100)
rwmixpct=100
#Random number algorithm for "--blockvarpct". Values: "fast" for high speed but weaker randomness; "balanced" for good balance of speed and randomness; "str>
blockvaralgo=balanced
#Block variance percentage. Defines the percentage of each block that will be refilled with random data between writes. This can be used to defeat compressi>
blockvarpct=100' WHERE name = '4k_100read_100random';
UPDATE benchmark_profiles SET config_json = '# Fichier d''exmple de profil pour un benchmark via l''outil elbencho.
# Pour une liste des commandes disponnibles utiliser la commande elbencho --help-all
# Profil: 8k_50read_100random = Test d''émulation BDD Online Transactional Processing (OLTP) (mise à jour, insertion, suppression)

#BASE
#Number of I/O worker threads. (Default: 1)
threads=8
#Depth of I/O queue per thread for asynchronous I/O. Setting this to 2 or higher turns on async I/O. (Default: 1)
iodepth=4
#Time limit in seconds for each benchmark phase. If the limit is exceeded for a phase then no further phases will run. (Default: 0 for disabled)
timelimit=300
#Number of bytes to read/write in a single operation. Each thread needs to keep one block in RAM (or multiple blocks if "--iodepth" is used), so be careful with large block sizes. (Default: 1M; supports base2 suffixes, e.g. "128K")
block=8k
#Use direct IO (also known as O_DIRECT) to avoid file contents caching. Note: For network or cluster filesystems, it depends on the actual filesystem whether this option is only effective for the client-side cache or also for the server-side cache. Also, some filesystems might ignore this completely or might have a RAID controller cache which operates independent of this setting.
direct=1
#Let I/O threads run in an infinite loop, i.e. they restart from the beginning when the reach the end of the specified workload. Terminate this via ctrl+c or by using "--timelimit"
infloop=1
#Preallocate file disk space in a write phase via posix_fallocate().
preallocfile=1
#Benchmark paths are not shared between service instances. Thus, each service instance will work on its own full dataset instead of a fraction of the data set.
nosvcshare=1


#AFFICHAGE
#Show minimum, average and maximum latency for read/write operations and entries. In read and write phases, entry latency includes file open, read/write and>
lat=1
#Show CPU utilization in phase stats results.
cpu=1
#Show latency histogram.
lathisto=0


#PROFILE
#Read files.
#read=1
#Write files. Create them if they don''t exist.
write=1
#Read/write at random offsets.
rand=1
#Random number algorithm for "--rand". Values: "fast" for high speed but weaker randomness; "balanced_single" for good balance of speed and randomness; "strong" for high CPU cost but strong randomness. (Default: a special algo for maximum single pass block coverage in write phase for aligned IO and "balanced_single" for reads and unaligned IO)
randalgo=balanced_single
#Percentage of blocks that should be read in a write phase. (Default: 0; Max: 100)
rwmixpct=50
#Random number algorithm for "--blockvarpct". Values: "fast" for high speed but weaker randomness; "balanced" for good balance of speed and randomness; "str>
blockvaralgo=balanced
#Block variance percentage. Defines the percentage of each block that will be refilled with random data between writes. This can be used to defeat compressi>
blockvarpct=100' WHERE name = '8k_50read_100random';
UPDATE benchmark_profiles SET config_json = '# Fichier d''exmple de profil pour un benchmark via l''outil elbencho.
# Pour une liste des commandes disponnibles utiliser la commande elbencho --help-all
# Profil: 256k_0read_0random = Ecriture intensive gros blocs (test extrême des 4 coins)

#BASE
#Number of I/O worker threads. (Default: 1)
threads=8
#Depth of I/O queue per thread for asynchronous I/O. Setting this to 2 or higher turns on async I/O. (Default: 1)
iodepth=4
#Time limit in seconds for each benchmark phase. If the limit is exceeded for a phase then no further phases will run. (Default: 0 for disabled)
timelimit=300
#Number of bytes to read/write in a single operation. Each thread needs to keep one block in RAM (or multiple blocks if "--iodepth" is used), so be careful with large block sizes. (Default: 1M; supports base2 suffixes, e.g. "128K")
block=256k
#se direct IO (also known as O_DIRECT) to avoid file contents caching. Note: For network or cluster filesystems, it depends on the actual filesystem whether this option is only effective for the client-side cache or also for the server-side cache. Also, some filesystems might ignore this completely or might have a RAID controller cache which operates independent of this setting.
direct=1
#Let I/O threads run in an infinite loop, i.e. they restart from the beginning when the reach the end of the specified workload. Terminate this via ctrl+c or by using "--timelimit"
infloop=1
#Preallocate file disk space in a write phase via posix_fallocate().
preallocfile=1
#Benchmark paths are not shared between service instances. Thus, each service instance will work on its own full dataset instead of a fraction of the data set.
nosvcshare=1


#AFFICHAGE
#Show minimum, average and maximum latency for read/write operations and entries. In read and write phases, entry latency includes file open, read/write and>
lat=1
#Show CPU utilization in phase stats results.
cpu=1
#Show latency histogram.
lathisto=0


#PROFILE
#Read files.
#read=1
write=1
#Read/write at random offsets.
rand=0
#Random number algorithm for "--rand". Values: "fast" for high speed but weaker randomness; "balanced_single" for good balance of speed and randomness; "strong" for high CPU cost but strong randomness. (Default: a special algo for maximum single pass block coverage in write phase for aligned IO and "balanced_single" for reads and unaligned IO)
randalgo=balanced_single
#Percentage of blocks that should be read in a write phase. (Default: 0; Max: 100)
rwmixpct=0
#Random number algorithm for "--blockvarpct". Values: "fast" for high speed but weaker randomness; "balanced" for good balance of speed and randomness; "str>
blockvaralgo=balanced
#Block variance percentage. Defines the percentage of each block that will be refilled with random data between writes. This can be used to defeat compressi>
blockvarpct=100' WHERE name = '256k_0read_0random';
UPDATE benchmark_profiles SET config_json = '# Fichier d''exmple de profil pour un benchmark via l''outil elbencho.
# Pour une liste des commandes disponnibles utiliser la commande elbencho --help-all
# Profil: 256k_100read_0random = Lecture intensive gros blocs (test extrême des 4 coins)

#BASE
#Number of I/O worker threads. (Default: 1)
threads=8
#Depth of I/O queue per thread for asynchronous I/O. Setting this to 2 or higher turns on async I/O. (Default: 1)
iodepth=4
#Time limit in seconds for each benchmark phase. If the limit is exceeded for a phase then no further phases will run. (Default: 0 for disabled)
timelimit=300
#Number of bytes to read/write in a single operation. Each thread needs to keep one block in RAM (or multiple blocks if "--iodepth" is used), so be careful with large block sizes. (Default: 1M; supports base2 suffixes, e.g. "128K")
block=256k
#se direct IO (also known as O_DIRECT) to avoid file contents caching. Note: For network or cluster filesystems, it depends on the actual filesystem whether this option is only effective for the client-side cache or also for the server-side cache. Also, some filesystems might ignore this completely or might have a RAID controller cache which operates independent of this setting.
direct=1
#Let I/O threads run in an infinite loop, i.e. they restart from the beginning when the reach the end of the specified workload. Terminate this via ctrl+c or by using "--timelimit"
infloop=1
#Preallocate file disk space in a write phase via posix_fallocate().
preallocfile=1
#Benchmark paths are not shared between service instances. Thus, each service instance will work on its own full dataset instead of a fraction of the data set.
nosvcshare=1


#AFFICHAGE
#Show minimum, average and maximum latency for read/write operations and entries. In read and write phases, entry latency includes file open, read/write and>
lat=1
#Show CPU utilization in phase stats results.
cpu=1
#Show latency histogram.
lathisto=0


#PROFILE
#Read files.
read=1
#write=1
#Read/write at random offsets.
rand=0
#Random number algorithm for "--rand". Values: "fast" for high speed but weaker randomness; "balanced_single" for good balance of speed and randomness; "strong" for high CPU cost but strong randomness. (Default: a special algo for maximum single pass block coverage in write phase for aligned IO and "balanced_single" for reads and unaligned IO)
randalgo=balanced_single
#Percentage of blocks that should be read in a write phase. (Default: 0; Max: 100)
rwmixpct=0
#Random number algorithm for "--blockvarpct". Values: "fast" for high speed but weaker randomness; "balanced" for good balance of speed and randomness; "str>
blockvaralgo=balanced
#Block variance percentage. Defines the percentage of each block that will be refilled with random data between writes. This can be used to defeat compressi>
blockvarpct=100' WHERE name = '256k_100read_0random';
