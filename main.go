package main

import (
	"encoding/binary"
	"log"

	"periph.io/x/conn/v3/i2c"
	"periph.io/x/host/v3"
	"periph.io/x/host/v3/i2c/i2creg"
)

func main() {
	// Initialize periph.io
	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}

	// Open the I2C bus
	bus, err := i2creg.Open("")
	if err != nil {
		log.Fatal(err)
	}
	defer bus.Close()

	// PMBus device address
	const addr = 0x40
	dev := &i2c.Dev{Addr: addr, Bus: bus}

	// PMBus command to set VOUT: 0x21
	const VOUT_COMMAND = 0x21

	// Convert 1.8V to LINEAR16 format
	// V = N × 2^E → For LINEAR16: exponent = -9, mantissa = 1.8 / 2^-9 = 921.6 ≈ 922
	// So LINEAR16 value is: exponent = -9 → 0xFFF7, mantissa = 922 → 0x039A
	linear16 := uint16(0xF739A) // 16-bit word: high byte = exponent, low = mantissa

	// But we need to split it into two bytes (LSB first)
	buf := make([]byte, 3)
	buf[0] = VOUT_COMMAND
	binary.LittleEndian.PutUint16(buf[1:], linear16)

	// Send the PMBus command
	if err := dev.Tx(buf, nil); err != nil {
		log.Fatalf("Failed to write VOUT_COMMAND: %v", err)
	}

	log.Println("VOUT set to 1.8V")
}

