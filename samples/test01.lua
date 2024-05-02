local vars = require "./vars/env"
local data = require "./components/data"

return {
    result = {
        test = testAdd(1,2),
    },
    data = data,
    environment = vars.environment,
    envTest = vars.test()
}