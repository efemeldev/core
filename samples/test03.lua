local data = require "./components/data"
local vars = require "./vars/env"

local function someTest()
    return "some test value"
end

return function() 
    return function()
        return {
            result = data,
            vars = vars,
            fnVal = someTest
        }
    end
end

-- return {
--     result = {
--         test = testAdd(1,2),
--     },
--     data = data,
--     environment = vars.environment
-- }