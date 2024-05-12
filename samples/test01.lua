local vars = require "./vars/env"
local data = require "samples/components/data"

print("test01", _EFEMEL_SCRIPT_PATH)

return {
    result = {
        test = testAdd(1,2),
    },
    data = data,
    environment = vars.environment,
    envTest = vars.test(),
    value1 = vars.recursive.value1,
    value2 = vars.recursive.value2,
}