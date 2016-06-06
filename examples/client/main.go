package main

import (
	"crypto/tls"

	"github.com/opsee/gmunch/client"
	"github.com/spf13/viper"
	"google.golang.org/grpc/credentials"
)

func main() {
	viper.SetEnvPrefix("gmunch")
	viper.AutomaticEnv()

	client, err := client.New(viper.GetString("address"), credentials.NewTLS(&tls.Config{
		InsecureSkipVerify: true,
	}))
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
