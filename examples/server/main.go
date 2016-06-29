package main

import (
	"os"
	"os/signal"
	"syscall"

	log "github.com/opsee/logrus"
	"github.com/opsee/gmunch"
	consumer "github.com/opsee/gmunch/consumer/kinesis"
	"github.com/opsee/gmunch/examples/debug"
	producer "github.com/opsee/gmunch/producer/kinesis"
	"github.com/opsee/gmunch/server"
	"github.com/opsee/gmunch/worker"
	"github.com/spf13/viper"
)

func main() {
	viper.SetEnvPrefix("gmunch")
	viper.AutomaticEnv()

	server := server.New(server.Config{
		LogLevel: viper.GetString("log_level"),
		Producer: producer.New(producer.Config{
			Stream: viper.GetString("kinesis_stream"),
		}),
		Consumer: consumer.New(consumer.Config{
			Stream:        viper.GetString("kinesis_stream"),
			EtcdEndpoints: viper.GetStringSlice("etcd_address"),
			ShardPath:     viper.GetString("shard_path"),
		}),
		Dispatch: worker.Dispatch{
			"test_event": func(evt *gmunch.Event) []worker.Task {
				return []worker.Task{debug.New(evt)}
			},
		},
	})

	sigChan := make(chan os.Signal, 1)
	errChan := make(chan error)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		errChan <- server.Start(
			viper.GetString("address"),
			viper.GetString("cert"),
			viper.GetString("cert_key"),
		)
	}()

	var err error
	select {
	case err = <-errChan:
		log.Info("received error from grpc service")
	case <-sigChan:
		log.Info("received interrupt")
	}

	server.Stop()

	if err != nil {
		log.Fatal(err)
	}
}
