package redis

import goredis "github.com/redis/go-redis/v9"

var renameField = goredis.NewScript(`
local v = redis.call('HGET', KEYS[1], ARGV[1])
if not v then return redis.error_reply('PLUGBOARD_CHANGED') end
if ARGV[1] == ARGV[2] then return 1 end
if redis.call('HEXISTS', KEYS[1], ARGV[2]) == 1 then return redis.error_reply('PLUGBOARD_EXISTS') end
redis.call('HSET', KEYS[1], ARGV[2], v)
redis.call('HDEL', KEYS[1], ARGV[1])
return 1`)

var listSet = goredis.NewScript(`
if redis.call('LINDEX', KEYS[1], ARGV[1]) ~= ARGV[2] then return redis.error_reply('PLUGBOARD_CHANGED') end
redis.call('LSET', KEYS[1], ARGV[1], ARGV[3])
return 1`)

var listRemove = goredis.NewScript(`
if redis.call('LINDEX', KEYS[1], ARGV[1]) ~= ARGV[2] then return redis.error_reply('PLUGBOARD_CHANGED') end
redis.call('LSET', KEYS[1], ARGV[1], ARGV[3])
redis.call('LREM', KEYS[1], 1, ARGV[3])
return 1`)

var replaceMember = goredis.NewScript(`
if redis.call('SISMEMBER', KEYS[1], ARGV[1]) == 0 then return redis.error_reply('PLUGBOARD_CHANGED') end
if ARGV[1] == ARGV[2] then return 1 end
if redis.call('SISMEMBER', KEYS[1], ARGV[2]) == 1 then return redis.error_reply('PLUGBOARD_EXISTS') end
redis.call('SREM', KEYS[1], ARGV[1])
redis.call('SADD', KEYS[1], ARGV[2])
return 1`)

var renameMember = goredis.NewScript(`
local s = redis.call('ZSCORE', KEYS[1], ARGV[1])
if not s then return redis.error_reply('PLUGBOARD_CHANGED') end
if ARGV[1] == ARGV[2] then return 1 end
if redis.call('ZSCORE', KEYS[1], ARGV[2]) then return redis.error_reply('PLUGBOARD_EXISTS') end
redis.call('ZREM', KEYS[1], ARGV[1])
redis.call('ZADD', KEYS[1], s, ARGV[2])
return 1`)
