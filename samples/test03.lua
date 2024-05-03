local data = require "./components/data"
local vars = require "./vars/env"

return function() 
    return function()
        return {
            result = data,
            vars = vars
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