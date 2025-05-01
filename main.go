/* Author: Austin Sierra
*  Company: FutureBit
 */

package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"math"
	"os"

	"golang.org/x/sys/unix"
)

var (
	// Command-line flags
	i2cBusPath    = flag.String("bus", "/dev/i2c-0", "I2C bus path")
	pmbusAddress  = flag.Int("addr", 0x40, "PMBus address of SiC45x device")
	targetVoltage = flag.Float64("volt", 1.0, "Target output voltage in volts") // target volts to 1
)

const (
	VOUT_COMMAND = 0x21 //page 13
	READ_VOUT    = 0x8B //PMBus COMMAND LIST page 16
	READ_IOUT    = 0x8C
	I2C_SLAVE    = 0x0703 //Set the address of the slave device
)

//assume operations are atomic and uninterrupted
func main() {
	flag.Parse() // Parse the command-line flags and check for valid targets
	if *pmbusAddress < 0 || *pmbusAddress > 0x7F {
		log.Fatalf("Invalid PMBus address: must be 0x00 to 0x7F, got 0x%X", *pmbusAddress)
	}
	if *targetVoltage < 0.3 || *targetVoltage > 5.0 {
		log.Fatalf("Invalid target voltage: must be 0.3V to 5.0V, got %.3fV", *targetVoltage)
	}

	// Open I2C bus that appear as device files under /dev/.
	// 0600 This is the file permission mode,
	file, err := os.OpenFile(*i2cBusPath, os.O_RDWR, 0600)
	if err != nil {
		log.Fatalf("Failed to open I2C bus: %v", err)
	}
	defer file.Close()

	fd := int(file.Fd()) //return file descriptor volume

	//PMBus ADDRESS (ADDR pin on page 11)
	/*The SiC45x has a 7-bit register that are used to set the base
	PMBus address of the device. A resistor assembled
	between ADDR pin and ground sets an offset from the
	default pre-configured MFR base address in the memory.
	Up to 15 different offsets can be set allowing 15 SiC45x
	devices with unique addresses in a single system. This
	offset and therefore the device address is read by the ADC
	during the initialization sequence. The table below provides
	the resistor values needed to set the 15 offsets from the
	base address. Please do not leave the setting resistor open
	or short.*/
	// Bind to device address using ioctl
	//The PMBus address is not hardcoded.
	//Instead, it's derived as:
	//PMBus address = BASE_ADDRESS + OFFSET
	if err := unix.IoctlSetInt(fd, I2C_SLAVE, *pmbusAddress); err != nil {
		log.Fatalf("Failed to set I2C address: %v", err)
	}

	// Set voltage
	if err := initDCandSetVoltage(fd, *targetVoltage); err != nil {
		log.Fatalf("Set voltage error: %v", err)
	}

	// Read voltage
	if volts, err := readPMBusLinear11(fd, READ_VOUT); err == nil {
		fmt.Printf("Voltage: %.3f V\n", volts)
	} else {
		log.Fatalf("Read voltage error: %v", err)
	}

	// Read current
	if amps, err := readPMBusLinear11(fd, READ_IOUT); err == nil {
		fmt.Printf("Current: %.3f A\n", amps)
	} else {
		log.Fatalf("Read current error: %v", err)
	}
}

func initDCandSetVoltage(fd int, targetVoltage float64) error {
	data := floatToLinear11(targetVoltage)
	packet := []byte{VOUT_COMMAND, data[0], data[1]}
	_, err := unix.Write(fd, packet)
	return err
}

func readPMBusLinear11(fd int, command byte) (float64, error) {
	// Send read command
	if _, err := unix.Write(fd, []byte{command}); err != nil {
		return 0, err
	}

	// Read 2 bytes of data
	buf := make([]byte, 2)
	if _, err := unix.Read(fd, buf); err != nil {
		return 0, err
	}

	return linear11ToFloat(buf), nil
}

/*
This function is used to convert the floating-point voltage value (like 3.3V, 5V, etc.)
into a integer format that can be transmitted via I2C or another protocol.
*/
func floatToLinear11(volts float64) [2]byte {
	var exponent int
	var mantissa int
	// Try different exponents to fit mantissa in 11-bit signed range (-1024 to 1023)
	for exponent = -15; exponent < 16; exponent++ {
		m := volts / math.Pow(2, float64(exponent))
		if m >= -1024 && m < 1024 {
			mantissa = int(math.Round(m))
			break
		}
	}
	// Compose 16-bit value: upper 5 bits are exponent, lower 11 bits are mantissa
	raw := uint16((int16(exponent)&0x1F)<<11) | uint16(uint16(mantissa)&0x07FF)
	return [2]byte{byte(raw & 0xFF), byte((raw >> 8) & 0xFF)}
}

/*
This function is the opposite of the previous
*/
func linear11ToFloat(b []byte) float64 {
	raw := binary.BigEndian.Uint16(b)
	exp := int8(raw >> 11)
	if exp > 15 {
		exp -= 32
	}
	mant := int16(raw & 0x07FF)
	if mant > 1023 {
		mant -= 2048
	}
	return float64(mant) * math.Pow(2, float64(exp))

}
