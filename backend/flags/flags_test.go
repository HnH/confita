package flags

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/HnH/confita"
	"github.com/HnH/confita/backend"
	"github.com/stretchr/testify/require"
)

type Config struct {
	A    string        `config:"a"`
	Adef string        `config:"a-def,short=ad"`
	B    bool          `config:"b"`
	Bdef bool          `config:"b-def,short=bd"`
	C    time.Duration `config:"c"`
	Cdef time.Duration `config:"c-def,short=cd"`
	D    int           `config:"d"`
	Ddef int           `config:"d-def,short=dd"`
	E    uint          `config:"e"`
	Edef uint          `config:"e-def,short=ed"`
	F    float32       `config:"f"`
	Fdef float32       `config:"f-def,short=fd"`
}

func runHelper(t *testing.T, args ...string) *Config {
	t.Helper()

	var output bytes.Buffer

	cs := []string{"-test.run=TestHelperProcess", "--"}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Stderr = &output
	cmd.Env = []string{"GO_HELPER_PROCESS=1"}
	err := cmd.Run()
	require.NoError(t, err)

	var cfg Config

	err = json.NewDecoder(&output).Decode(&cfg)
	require.NoError(t, err)

	return &cfg
}

func TestFlags(t *testing.T) {
	t.Run("Use defaults", func(t *testing.T) {
		cfg := runHelper(t, "-a=hello", "-b=true", "-c=10s", "-d=-100", "-e=1", "-f=100.01")
		require.Equal(t, "hello", cfg.A)
		require.Equal(t, true, cfg.B)
		require.Equal(t, 10*time.Second, cfg.C)
		require.Equal(t, -100, cfg.D)
		require.Equal(t, uint(1), cfg.E)
		require.Equal(t, float32(100.01), cfg.F)
	})

	t.Run("Override defaults", func(t *testing.T) {
		cfg := runHelper(t, "-a-def=bye", "-b-def=false", "-c-def=15s", "-d-def=-200", "-e-def=400", "-f-def=2.33")
		require.Equal(t, "bye", cfg.Adef)
		require.Equal(t, false, cfg.Bdef)
		require.Equal(t, 15*time.Second, cfg.Cdef)
		require.Equal(t, -200, cfg.Ddef)
		require.Equal(t, uint(400), cfg.Edef)
		require.Equal(t, float32(2.33), cfg.Fdef)
	})
}

func TestFlagsShort(t *testing.T) {
	cfg := runHelper(t, "-ad=hello", "-bd=true", "-cd=20s", "-dd=500", "-ed=700", "-fd=333.33")
	require.Equal(t, "hello", cfg.Adef)
	require.Equal(t, true, cfg.Bdef)
	require.Equal(t, 20*time.Second, cfg.Cdef)
	require.Equal(t, 500, cfg.Ddef)
	require.Equal(t, uint(700), cfg.Edef)
	require.Equal(t, float32(333.33), cfg.Fdef)
}

func TestFlagsMixed(t *testing.T) {
	cfg := runHelper(t, "-ad=hello", "-b-def=true", "-cd=20s", "-d-def=500", "-ed=600", "-f-def=42.42")
	require.Equal(t, "hello", cfg.Adef)
	require.Equal(t, true, cfg.Bdef)
	require.Equal(t, 20*time.Second, cfg.Cdef)
	require.Equal(t, 500, cfg.Ddef)
	require.Equal(t, uint(600), cfg.Edef)
	require.Equal(t, float32(42.42), cfg.Fdef)
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_HELPER_PROCESS") != "1" {
		return
	}

	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "No args\n")
		os.Exit(2)
	}

	os.Args = append(os.Args[:1], args...)

	cfg := Config{
		Adef: "hello",
		Bdef: true,
		Cdef: 10 * time.Second,
		Ddef: -100,
	}

	err := confita.NewLoader(NewBackend()).Load(context.Background(), &cfg)
	require.NoError(t, err)
	err = json.NewEncoder(os.Stderr).Encode(&cfg)
	require.NoError(t, err)
	os.Exit(0)
}

type store map[string]string

func (s store) Get(ctx context.Context, key string) ([]byte, error) {
	data, ok := s[key]
	if !ok {
		return nil, backend.ErrNotFound
	}

	return []byte(data), nil
}

func (store) Name() string {
	return "store"
}

func TestWithAnotherBackend(t *testing.T) {
	s := struct {
		String   string        `config:"string,required"`
		Bool     bool          `config:"bool,required"`
		Int      int           `config:"int,required"`
		Uint     uint          `config:"uint,required"`
		Float    float64       `config:"float,required"`
		Duration time.Duration `config:"duration,required"`
	}{}

	st := store{
		"string":   "string",
		"bool":     "true",
		"int":      "42",
		"uint":     "42",
		"float":    "42.42",
		"duration": "1ns",
	}

	err := confita.NewLoader(st, NewBackend()).Load(context.Background(), &s)
	require.NoError(t, err)
	require.Equal(t, "string", s.String)
	require.Equal(t, true, s.Bool)
	require.Equal(t, 42, s.Int)
	require.EqualValues(t, 42, s.Uint)
	require.Equal(t, 42.42, s.Float)
	require.Equal(t, time.Duration(1), s.Duration)
}
