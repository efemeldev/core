-- Define a custom Lua module
custom = {}

-- Define a function in the custom module that wraps around the global add function
function custom.add(a, b)
    return add(a, b)
end