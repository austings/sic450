## you may need prereq
go get golang.org/x/sys/unix

## then compile with flags and admin

-bus  // i2cBusPath specifies the I2C bus path to use for communication with the SiC45x device.

-addr // pmbusAddress specifies the PMBus address of the SiC45x device.

-volt //defaults to 1.0

sudo go run main.go -bus=/dev/i2c-1 -addr=0x42 -volt=1.0
