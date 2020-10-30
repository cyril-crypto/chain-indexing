package infrastructure_test

import (
	"time"

	"github.com/crypto-com/chainindex/infrastructure"
	"github.com/crypto-com/chainindex/internal/primptr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PgxConnPoolConfig", func() {
	Describe("ToURL", func() {
		Context("When no config is provided", func() {
			It("should return Postgres connection string without auth when no username is provided", func() {
				config := infrastructure.PgxConnPoolConfig{
					PgConnConfig: infrastructure.PgConnConfig{
						Host:          "127.0.0.1",
						Port:          5432,
						MaybeUsername: nil,
						MaybePassword: nil,
						Database:      "chainindex",
						SSL:           false,
					},
					MaybeMaxConns:          nil,
					MaybeMinConns:          nil,
					MaybeMaxConnLifeTime:   nil,
					MaybeMaxConnIdleTime:   nil,
					MaybeHealthCheckPeriod: nil,
				}

				Expect(config.ToURL()).To(Equal("postgres://127.0.0.1:5432/chainindex?sslmode=disable"))
			})

			It("should return Postgres connection string without auth when username and password is provided", func() {
				config := infrastructure.PgxConnPoolConfig{
					PgConnConfig: infrastructure.PgConnConfig{
						Host:          "127.0.0.1",
						Port:          5432,
						MaybeUsername: primptr.String("user"),
						MaybePassword: primptr.String("password"),
						Database:      "chainindex",
						SSL:           false,
					},
					MaybeMaxConns:          nil,
					MaybeMinConns:          nil,
					MaybeMaxConnLifeTime:   nil,
					MaybeMaxConnIdleTime:   nil,
					MaybeHealthCheckPeriod: nil,
				}

				Expect(config.ToURL()).To(Equal("postgres://user:password@127.0.0.1:5432/chainindex?sslmode=disable"))
			})

			It("should return Postgres connection string when SSL is enabled", func() {
				config := infrastructure.PgxConnPoolConfig{
					PgConnConfig: infrastructure.PgConnConfig{
						Host:          "127.0.0.1",
						Port:          5432,
						MaybeUsername: primptr.String("user"),
						MaybePassword: primptr.String("password"),
						Database:      "chainindex",
						SSL:           true,
					},
					MaybeMaxConns:          nil,
					MaybeMinConns:          nil,
					MaybeMaxConnLifeTime:   nil,
					MaybeMaxConnIdleTime:   nil,
					MaybeHealthCheckPeriod: nil,
				}

				Expect(config.ToURL()).To(Equal("postgres://user:password@127.0.0.1:5432/chainindex"))
			})
		})

		Context("When config is provided", func() {
			It("should render the config in the connection string", func() {
				config := infrastructure.PgxConnPoolConfig{
					PgConnConfig: infrastructure.PgConnConfig{
						Host:          "127.0.0.1",
						Port:          5432,
						MaybeUsername: primptr.String("user"),
						MaybePassword: primptr.String("password"),
						Database:      "chainindex",
						SSL:           true,
					},
					MaybeMaxConns:          primptr.Int32(50),
					MaybeMinConns:          primptr.Int32(1),
					MaybeMaxConnLifeTime:   primptr.Duration(300 * time.Second),
					MaybeMaxConnIdleTime:   primptr.Duration(60 * time.Second),
					MaybeHealthCheckPeriod: primptr.Duration(30 * time.Second),
				}

				Expect(config.ToURL()).To(Equal("postgres://user:password@127.0.0.1:5432/chainindex?pool_health_check_period=30s&pool_max_conn_idle_time=1m0s&pool_max_conn_lifetime=5m0s&pool_max_conns=50&pool_min_conns=1"))
			})
		})
	})
})