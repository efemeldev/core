local data = require "./components/data"

return {
    result = {
        add = custom.add(2,55),
        test = testAdd(1,2),
    },
    data = data,
    environment = vars.environment,
    path = package.path
}