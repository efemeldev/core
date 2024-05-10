local data = require "samples/components/data"
local vars = require "samples/vars/env"

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