-- https://github.com/wg/wrk
-- wrk -c 1000 -d 10s -s wrk-test.lua http://localhost:8080

math.randomseed(os.time())
randomIP = function()
	return
	math.random(1,254) .. "." ..
	math.random(1,254) .. "." ..
	math.random(1,254) .. "." ..
	math.random(1,254)
end

request = function()
	path = "/json/" .. randomIP()
	return wrk.format(nil, path)
end
