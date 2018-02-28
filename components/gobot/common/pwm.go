package common

// PWMDriver defines PWM Driver
type PWMDriver interface {
	// SetPWMFrequency sets PWM frequency Hz
	SetPWMFrequency(uint) error
	// SetPWMPulse generates a pulse on channel
	SetPWMPulse(chn int, on uint, off uint) error
}
