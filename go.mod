module strike

go 1.12

require (
	git.internal.yunify.com/MDMP2/cloudevents v0.0.0-20190417033153-b26740c15a29
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20190321100706-95778dfbb74e // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/dgryski/go-farm v0.0.0-20190423205320-6a90982ecee2 // indirect
	github.com/google/btree v1.0.0 // indirect
	github.com/google/uuid v1.1.1
	github.com/kavu/go_reuseport v1.4.0
	github.com/olebedev/emitter v0.0.0-20190110104742-e8d1457e6aee // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/rcrowley/go-metrics v0.0.0-20181016184325-3113b8401b8a
	github.com/sirupsen/logrus v1.4.1 // indirect
	github.com/smartystreets/goconvey v0.0.0-20190330032615-68dc04aab96a // indirect
	github.com/ugorji/go v1.1.4 // indirect
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4
	qingcloud.com/qing-cloud-mq v0.0.0-00010101000000-000000000000
)

replace qingcloud.com/qing-cloud-mq => ../qingcloud.com/qing-cloud-mq/
