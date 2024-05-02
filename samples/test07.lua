local data = require "./components/data"
local vars = require "./vars/env"

return {
    result = {
        test = testAdd(1,2),
    },
    data = data,
    environment = vars.environment
}