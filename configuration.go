package aarhusboligventeliste

import (
    	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
"log"
)

type Config struct {
    Username string
    Password string
}

const (
    ConfigEntityType = "configuration"
    ConfigKey = "ConfigurationKey"
)

func GetConfig(ctx context.Context) Config{
	var conf Config;
    log.Printf("getting conf %v", conf)

    confKey := datastore.NewKey(ctx, ConfigEntityType, ConfigKey, 0, nil)

	// fill
	if err := datastore.Get(ctx, confKey, &conf); err != nil {
        log.Printf("err %v", err)
		if err == datastore.ErrNoSuchEntity {
            //First time we do not have a configuration we add a dummy one
			conf.Username = "foo"
            conf.Password = "bar"
			if _, err := datastore.Put(ctx, confKey, &conf); err != nil {
                log.Printf("err %v", err)
                panic(err)
            }
			return conf
		}
		panic("cannot load builder key: " + err.Error())
	}
    log.Printf("returning conf %v", conf)

	return conf
}