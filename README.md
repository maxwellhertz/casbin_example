# Tutorial: Integrate Gin with Cabsin

I've been working on a Java project recently in which our team uses [Apache Shiro](http://shiro.apache.org/) to do some authentication and authorization stuff. Now I just wonder if there exists a framework like Shiro in Go's world? After some search, I found [Casbin](https://github.com/casbin/casbin). It's also a powerful and interesting library but maybe it's kinda hard for a newbie to write a demo from sractch. 

I completed a basic web app based on [Gin](https://github.com/gin-gonic/gin) and Casbin after struggling for a whole afternoon. This tutorial is a simple replay. Hope it will help. :)

## Structure of Our Project

```
root/
    main.go              # entry point of application                       
    handler/             # Gin handler functions
    middleware           # Gin middlewares
    config/              # some configuration files like Casbin's rbac_model.conf
    component/           # global components like GORM DB instance 
```

## Initialize DB Connection And Cache

Create a file called `persistence.go` in `component` in which we will initialize DB connection using [GORM](https://github.com/jinzhu/gorm) and cache using [BigCache](https://github.com/allegro/bigcache).

```go
import (
	"fmt"
	"github.com/allegro/bigcache"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"time"
)

var (
	DB           *gorm.DB
	GlobalCache  *bigcache.BigCache
)

func init() {
	// Connect to DB
	var err error
	DB, err = gorm.Open("mysql", "your_db_url")
	if err != nil {
		panic(fmt.Sprintf("failed to connect to DB: %v", err))
	}

	// Initialize cache
	GlobalCache, err = bigcache.NewBigCache(bigcache.DefaultConfig(30 * time.Minute)) // Set expire time to 30 mins
	if err != nil {
		panic(fmt.Sprintf("failed to initialize cahce: %v", err))
	}
}
```

In our application, we will store Casbin's policies in DB (which we will talk about soon) and store current user in cache.

## Configure Casbin

### Model Configuration File

Ar first you may find some concepts in Casbin quite confusing. The first one is its model configuration file. I don't want to talk too much about it here (cuz I don't get it very well yet :persevere:) so I'm gonna give a simple example which is quite specific to our application. We will control a user's request based on 
his role, which is called RBAC aka Role-based access control. Therefore we will create a `rbac_model.conf` in `configuration` directory.

```conf
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act
```

A model configuration file tells Casbin how to determine if a user has some qualifications. In the above example, we just declares some denifitions:

1. `r = sub, obj, act` defines that a limited request will be consisted of three parts: *sub*ject - user, *obj*ect - URL or more generally resource and *act*ion - operation.
2. `p = sub, obj, act` defines the format of a policy. For example, `admin, data, write` means `All admins can write data.`
3. `e = some(where (p.eft == allow))` means that a user can do something as long as there is a defined policy which allows him to do so.
4. `g = _, _` defines the format of definition of user's role. For example, `Alice, admin` indicates Alice is an admin.
5. `m = g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act` defines the workflow of authorization: check user's role -> check the resource which user is trying to access -> check the action of user. 

Note that there are at least four sections in a model configuration file: `request_definition`, `policy_definition`, `policy_effect` and `matchers`. Sometimes we don't do RBAC so `role_definition` section is not necessary.

### Policies

Let's say we have some policies and user groups like this:

```csv
p, user, data, read
p, admin, data, read
p, admin, data, write
g, Alice, admin
g, Bob, user
```

Here we firstly define three policies: 

1. All users can only read data.
2. All admins can read data.
3. All admins can also write data. 

Then we assign roles to user:

1. Alice is an admin.
2. Bob is a user.

Thus Alice has full control over data1 while Bob can only read data1. He will be blocked if he wants to write data1.

Casbin allows us to simplily store all policies in a CSV file and this is the most basic way. But this time we will store them in MySQL DB. Casbin stores policies in a table named `casbin_rule` and it will create this table automatically if not existed. In our case, the structure of table `casbin_rule` will look like this:

```sql
CREATE TABLE casbin_rule (
    p_type VARCHAR(100),
    v0 VARCHAR(100),
    v1 VARCHAR(100),
    v2 VARCHAR(100)
);
```

Add a policy:

```sql
INSERT INTO casbin_rule VALUES('p', 'user', 'data', 'read');
```

Add a user group:

```sql
INSERT INTO casbin_rule(p_type, v0, v1) VALUES('g', 'Bob', 'user');
```

## Implement Gin Handler Functions

At first we will implement the logic of user login. 

```go
// handler/user_handler.go

func Login(c *gin.Context) {
	username, password := c.PostForm("username"), c.PostForm("password")
    // Authentication
    // blahblah...

	// Generate random session id
	u, err := uuid.NewRandom()
	if err != nil {
		log.Fatal(err)
	}
	sessionId := fmt.Sprintf("%s-%s", u.String(), username)
	// Store current subject in cache
	component.GlobalCache.Set(sessionId, []byte(username))
	// Send cache key back to client in cookie
	c.SetCookie("current_subject", sessionId, 30*60, "/resource", "", false, true)
	c.JSON(200, component.RestResponse{Code: 1, Message:username + " logged in successfully"})
}
```

If a user has been identified, we need to store current user (or subject) in cache. In fact, what we do here is the same that Shiro stores current subject in session. Don't forget to send cache's key (or you can call it session id) back to client side.

Note that Shiro will do authentication stuff for us while Casbin just leaves that to us. So we have to implement authentication logic ourselves.

Don't forget to provide handlers for users to access resource:

```go
// handler/resource_handler.go

func ReadResource(c *gin.Context) {
    // some stuff
    // blahblah...

	c.JSON(200, component.RestResponse{Code: 1, Message: "read resource successfully", Data: "resource"})
}

func WriteResource(c *gin.Context) {
    // some stuff
    // blahblah...

	c.JSON(200, component.RestResponse{Code: 1, Message: "write resource successfully", Data: "resource"})
}
```

After this, we should register these functions and start our application in `main.go`:

```go
// main.go

var (
	router *gin.Engine
)

func init() {
	// Initialize gin router
	router = gin.Default()
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowCredentials = true
	router.Use(cors.New(corsConfig)) // CORS configuraion
    router.POST("/user/login", handler.Login)
    router.GET("/resource", handler.ReadResource)
    router.POST("/resource", handler.WriteResource)
}

func main() {
	defer component.DB.Close()

    // Start our application
	err := router.Run(":8081")
	if err != nil {
		panic(fmt.Sprintf("failed to start gin engin: %v", err))
	}
	log.Println("application is now running...")
}
```

Ok, almost done! The last piece and most important part of our application is secure our API by RBAC.

## Enforce Casbin Policies

### Load Policies From DB

The first problem is: how can we load policies from DB dynamically? We can do this using [Casbin Adapters](https://casbin.org/docs/en/adapters). More specifcally, we will use [Gorm Adapter](https://github.com/casbin/gorm-adapter) here. 

The first step is to initialize an adapter with existing GORM instance:

```go
// main.go

func init() {
	// Initialize  casbin adapter
	adapter, err := gormadapter.NewAdapterByDB(component.DB)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize casbin adapter: %v", err))
	}

	// Initialize gin router
	router = gin.Default()
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowCredentials = true
	router.Use(cors.New(corsConfig)) // CORS configuraion
    router.POST("/user/login", handler.Login)
    router.GET("/resource", handler.ReadResource)
    router.POST("/resource", handler.WriteResource)
}
```

Apparently, we should force policies to control access to resource before any relevant handler functions are called. In my opinion, an elegant way to do this is utilizing Gin's [middlewares](https://github.com/gin-gonic/gin#using-middleware) and [grouping routes](https://github.com/gin-gonic/gin#grouping-routes). Firstly, let's define a middleware in which our policies will be enforced. I think the code showed below is self-explanatory:

```go
// middleware/access_control.go

// Authorize determines if current subject has been authorized to take an action on an object.
func Authorize(obj string, act string, adapter *gormadapter.Adapter) gin.HandlerFunc {
	return func(c *gin.Context) {
        // Get current user/subject
		val, existed := c.Get("current_subject")
		if !existed {
			c.AbortWithStatusJSON(401, component.RestResponse{Message: "user hasn't logged in yet"})
			return
		}
		// Casbin enforces policy
		ok, err := enforce(val.(string), obj, act, adapter)
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(500, component.RestResponse{Message: "error occurred when authorizing user"})
			return
		}
		if !ok {
			c.AbortWithStatusJSON(403, component.RestResponse{Message: "forbidden"})
			return
		}
		c.Next()
	}
}

func enforce(sub string, obj string, act string, adapter *gormadapter.Adapter) (bool, error) {
    // Load model configuration file and policy store adapter
	enforcer, err := casbin.NewEnforcer("config/rbac_model.conf", adapter)
	if err != nil {
		return false, fmt.Errorf("failed to create casbin enforcer: %w", err)
	}
	// Load policies from DB dynamically
	err = enforcer.LoadPolicy()
	if err != nil {
		return false, fmt.Errorf("failed to load policy from DB: %w", err)
    }
    // Verify
	ok, err := enforcer.Enforce(sub, obj, act)
	return ok, err
}
```

At last, group all routes needed to secure and use our middleware:

```go
// main.go

func init() {
	// Initialize  casbin adapter
	adapter, err := gormadapter.NewAdapterByDB(component.DB)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize casbin adapter: %v", err))
	}

	// Initialize Gin router
	router = gin.Default()
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowCredentials = true
	router.Use(cors.New(corsConfig)) // CORS configuraion
    router.POST("/user/login", handler.Login)
    // Secure our API
	resource := router.Group("/api")
	{
		resource.GET("/resource", middleware.Authorize("resource", "read", adapter), handler.ReadResource)
		resource.POST("/resource", middleware.Authorize("resource", "write", adapter), handler.WriteResource)
	}
}
```

Boom! All are set to go! If a client doesn't log in firstly or he is not an admin, he will be denied when he tries to `GET /api/resource` or `POST /api/resource`.

[Source code]()

