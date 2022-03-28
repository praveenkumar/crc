package cmd

import (
	"github.com/code-ready/crc/pkg/crc/logging"
	"github.com/containers/gvisor-tap-vsock/pkg/types"
	"github.com/kardianos/service"
)

func runService(config *types.Configuration) error {
	svcConfig := &service.Config{}
	prg := &program{config: config}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		return err
	}

	errs := make(chan error, 5)
	logger, err := s.Logger(errs)
	if err != nil {
		return err
	}

	go func() {
		for {
			err := <-errs
			if err != nil {
				logging.Error(err)
			}
		}
	}()

	if err := s.Run(); err != nil {
		logger.Error(err)
	}
	return nil
}

type program struct {
	config *types.Configuration
	exit   chan struct{}
}

func (p *program) Start(s service.Service) error {
	if service.Interactive() {
		logging.Info("Running in terminal.")
	} else {
		logging.Info("Running under service manager.")
	}
	p.exit = make(chan struct{})

	// Start should not block. Do the actual work async.
	errCh := make(chan error, 1)
	go func() {
		err := run(p.config)
		if err != nil {
			errCh <- err
		}
	}()
	return nil
}

func (p *program) Stop(s service.Service) error {
	logging.Info("I'm Stopping!")
	close(p.exit)
	return nil
}
