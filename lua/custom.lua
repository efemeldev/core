-- Define a custom Lua module
custom = {
    test = 123
}

-- Define a function in the custom module that wraps around the global add function
function custom.add(a, b)
    -- print the arguments
    return a+b
end