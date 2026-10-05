#!lua name=my_utils

redis.register_function('delete_dialog_by_prefix', function(keys, args)
    local pattern = args[1]
    local count = args[2] or "250"
    local cursor = "0"
    local deleted_count = 0

    repeat
        local res = redis.call("SCAN", cursor, "MATCH", pattern, "COUNT", count)
        cursor = res[1]
        local found_keys = res[2]

        if #found_keys > 0 then
            local deleted = redis.call("UNLINK", unpack(found_keys))
            deleted_count = deleted_count + deleted
        end
    until cursor == "0"

    return deleted_count
end)
