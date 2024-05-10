local data = require "samples/components/data"
local vars = require "samples/vars/env"

return {
    result = {
        test = testAdd(1,2),
    },
    data = data,
    environment = vars.environment
}