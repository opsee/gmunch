package main

import (
	"crypto/tls"

	"github.com/opsee/gmunch/client"
	"github.com/spf13/viper"
)

func main() {
	viper.SetEnvPrefix("gmunch")
	viper.AutomaticEnv()

	config := client.Config{
		TLSConfig: tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client, err := client.New(viper.GetString("address"), config)
	if err != nil {
		panic(err)
	}

	err = client.Send("test_event", map[string]interface{}{
		"user_name":  "merk",
		"user_email": "compuper@merkmertin",
		"referrer":   "reddit",
	})

	if err != nil {
		panic(err)
	}
}
