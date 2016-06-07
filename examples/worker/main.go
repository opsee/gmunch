package main

import (
	"os"
	"os/signal"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/opsee/gmunch"
	consumer "github.com/opsee/gmunch/consumer/kinesis"
	"github.com/opsee/gmunch/examples/debug"
	"github.com/opsee/gmunch/worker"
	"github.com/spf13/viper"
)

func main() {
	viper.SetEnvPrefix("gmunch")
	viper.AutomaticEnv()

	worker := worker.New(worker.Config{
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
		errChan <- worker.Start()
	}()

	var err error
	select {
	case err = <-errChan:
		log.Info("received error from gmunch worker")
	case <-sigChan:
		log.Info("received interrupt")
	}

	worker.Stop()

	if err != nil {
		log.Fatal(err)
	}
}
