* Add Code game logic
* Add tests
* have concurrentgameslice to handle game removal reusing the removed slot
* check if it makes sense to add two new middlewares:
```
	// TODO:	router.Use(httprate.LimitByIP(100, 1*time.Minute))
	// TODO: 	router.Use(middleware.RequestSize(4096))
```