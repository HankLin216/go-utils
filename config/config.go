package config

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"time"

	"dario.cat/mergo"
	"go.uber.org/zap"

	// init encoding
	_ "github.com/HankLin216/go-utils/encoding/json"
	_ "github.com/HankLin216/go-utils/encoding/proto"
	_ "github.com/HankLin216/go-utils/encoding/yaml"
)

var _ Config = (*config)(nil)

var (
	ErrNotFound = errors.New("key not found") // ErrNotFound is key not found.
	logger      *zap.Logger
)

// Observer is config observer.
type Observer func(string, Value)

// Config is a config interface.
type Config interface {
	Load() error
	Scan(v interface{}) error
	Value(key string) Value
	Watch(key string, o Observer) error
	Close() error
	SetLogger(l *zap.Logger)
}

type config struct {
	opts      options
	reader    Reader
	cached    sync.Map
	observers sync.Map
	watchers  []Watcher
}

// New a config with options.
func New(opts ...Option) Config {
	o := options{
		decoder:  defaultDecoder,
		resolver: defaultResolver,
		merge: func(dst, src interface{}) error {
			return mergo.Map(dst, src, mergo.WithOverride)
		},
	}
	for _, opt := range opts {
		opt(&o)
	}

	logger, _ = zap.NewProduction()

	return &config{
		opts:   o,
		reader: newReader(o),
	}
}

func (c *config) SetLogger(l *zap.Logger) {
	logger = l
}

func (c *config) watch(w Watcher) {
	for {
		kvs, err := w.Next()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				logger.Info("watcher's ctx cancel", zap.Error(err))
				return
			}
			time.Sleep(time.Second)
			logger.Error("failed to watch next config", zap.Error(err))
			continue
		}
		if err := c.reader.Merge(kvs...); err != nil {
			logger.Error("failed to merge next config", zap.Error(err))
			continue
		}
		if err := c.reader.Resolve(); err != nil {
			logger.Error("failed to resolve next config", zap.Error(err))
			continue
		}
		c.cached.Range(func(key, value interface{}) bool {
			k := key.(string)
			v := value.(Value)
			if n, ok := c.reader.Value(k); ok && reflect.TypeOf(n.Load()) == reflect.TypeOf(v.Load()) && !reflect.DeepEqual(n.Load(), v.Load()) {
				v.Store(n.Load())
				if o, ok := c.observers.Load(k); ok {
					o.(Observer)(k, v)
				}
			}
			return true
		})
	}
}

func (c *config) Load() error {
	for _, src := range c.opts.sources {
		kvs, err := src.Load()
		if err != nil {
			return err
		}
		for _, v := range kvs {
			logger.Debug("config loaded", zap.String("key", v.Key), zap.String("format", v.Format))
		}
		if err = c.reader.Merge(kvs...); err != nil {
			logger.Error("failed to merge config source", zap.Error(err))
			return err
		}
		w, err := src.Watch()
		if err != nil {
			logger.Error("failed to watch config source", zap.Error(err))
			return err
		}
		c.watchers = append(c.watchers, w)
		go c.watch(w)
	}
	if err := c.reader.Resolve(); err != nil {
		logger.Error("failed to resolve config", zap.Error(err))
		return err
	}
	return nil
}

func (c *config) Value(key string) Value {
	if v, ok := c.cached.Load(key); ok {
		return v.(Value)
	}
	if v, ok := c.reader.Value(key); ok {
		c.cached.Store(key, v)
		return v
	}
	return &errValue{err: ErrNotFound}
}

func (c *config) Scan(v interface{}) error {
	data, err := c.reader.Source()
	if err != nil {
		return err
	}
	return unmarshalJSON(data, v)
}

func (c *config) Watch(key string, o Observer) error {
	if v := c.Value(key); v.Load() == nil {
		return ErrNotFound
	}
	c.observers.Store(key, o)
	return nil
}

func (c *config) Close() error {
	for _, w := range c.watchers {
		if err := w.Stop(); err != nil {
			return err
		}
	}
	return nil
}
