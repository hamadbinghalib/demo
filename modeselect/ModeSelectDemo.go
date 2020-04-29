package modeselect

import (
	"demo/ioexp"
	"demo/sensors"
	"demo/valves"
	"time"
)

// UserInput is a custome type struct that contains the global
// variables input by the user or operator
type UserInput struct {
	//Mode string
	//BreathType          string
	PatientTriggerType int
	TidalVolume        float32 // ml
	Rate               float32 // BPM
	Ti                 float32 // inhalation time
	Te                 float32 // exhalation time
	IR                 float32 // inhalation ratio part
	ER                 float32 // exhalation ratio part
	PeakFlow           float32
	PEEP               float32 // 5-20 mmH2O
	//FiO2                float32 // 21% - 100%
	PressureTrigSense float32 // -0.5 to 02 mmH2O
	//FlowTrigSense       float32 // 0.5 to 5 Lpm
	//PressureSupport     float32 // needs to be defined
	//InspiratoryPressure float32
	//PressureControl     float32
}

//Exit global var to control the system
var Exit bool

//ModeSelection ...s
func ModeSelection(UI *UserInput) {
	switch UI.PatientTriggerType {
	case 0:
		PressureControl(UI)
	case 1:
		PressureAssist(UI)
	default:
		PressureControl(UI)
	}
}

// UpdateValues ...
func UpdateValues(UI *UserInput) {
	BCT := 60 / UI.Rate
	if UI.Ti != 0 {
		UI.Te = BCT - UI.Ti
		UI.PeakFlow = (60 * UI.TidalVolume) / (UI.Ti * 1000)
	} else if UI.IR != 0 {
		UI.Ti = UI.IR / (UI.IR + UI.ER)
		UI.Te = BCT - UI.Ti
		UI.PeakFlow = (60 * UI.TidalVolume) / (UI.Ti * 1000)
	} else if UI.PeakFlow != 0 {
		UI.Ti = (60 * UI.TidalVolume) / (UI.PeakFlow * 1000)
		UI.Te = BCT - UI.Ti
	}
}

// PressureControl ...
func PressureControl(UI *UserInput) {
	//calculate Te from UI.Ti and BCT
	UpdateValues(UI)

	// Identify the main valves or solenoids by MIns and MExp
	MV := valves.SolenValve{Name: "A_PSV_INS", State: false, PinMask: ioexp.Solenoid0}   //normally closed
	MIns := valves.SolenValve{Name: "A_PSV_INS", State: false, PinMask: ioexp.Solenoid1} //normally closed
	MExp := valves.SolenValve{Name: "A_PSV_EXP", State: false, PinMask: ioexp.Solenoid2} //normally closed

	// Identify the flow sensors by PIns and PExp
	PExp := sensors.Pressure{Name: "SNS_P_EXP", ID: 1, AdcID: 1, MMH2O: 0} //expratory pressure sensor

	//control loop
	for !Exit {
		//Open main valve and MIns controlled
		for start := time.Now(); time.Since(start) < (time.Duration(UI.Ti*1000) * time.Millisecond); {
			MV.SolenCmd("Open")
			MIns.SolenCmd("Open")

		}
		//Close main valve and MIns
		MV.SolenCmd("Close")
		MIns.SolenCmd("Close") // closes the valve
		//Open main valve MExp controlled by flow sensor PExp
		for start := time.Now(); time.Since(start) < (time.Duration(UI.Te*1000) * time.Millisecond); {
			//safety measure
			if PExp.ReadPressure() <= UI.PEEP {
				break
			}
			MExp.SolenCmd("Open")
		}
		//Close main valve MExp
		MExp.SolenCmd("Close") // closes the valve
	}

}

// PressureAssist ...
func PressureAssist(UI *UserInput) {

	UpdateValues(UI)
	//Initialize  Sensors at inhalation side
	PIns := sensors.Pressure{Name: "SNS_P_INS", ID: 0, AdcID: 1, MMH2O: 0} //insparatory pressure sensor
	//Initialize  Sensors at exhalation side
	PExp := sensors.Pressure{Name: "SNS_P_EXP", ID: 2, AdcID: 1, MMH2O: 0} //expratory pressure sensor
	//Initialize valves
	MV := valves.SolenValve{Name: "A_PSV_INS", State: false, PinMask: ioexp.Solenoid0}   //normally closed
	MIns := valves.SolenValve{Name: "A_PSV_INS", State: false, PinMask: ioexp.Solenoid1} //normally closed
	MExp := valves.SolenValve{Name: "A_PSV_EXP", State: false, PinMask: ioexp.Solenoid2} //normally closed

	//Calculate trigger threshhold with PEEP and sensitivity
	PTrigger := UI.PEEP + UI.PressureTrigSense
	//Begin loop
	for !Exit {
		//check if trigger is true
		if PIns.ReadPressure() <= PTrigger {
			//Open main valve and MIns controlled
			for start := time.Now(); time.Since(start) < (time.Duration(UI.Ti*1000) * time.Millisecond); {
				MV.SolenCmd("Open")
				MIns.SolenCmd("Open")

			}
			//Close main valve and MIns
			MV.SolenCmd("Close")
			MIns.SolenCmd("Close") // closes the valve
			//Open main valve MExp controlled by flow sensor PExp
			for start := time.Now(); time.Since(start) < (time.Duration(UI.Te*1000) * time.Millisecond); {
				//safety measure
				if PExp.ReadPressure() <= UI.PEEP {
					break
				}
				MExp.SolenCmd("Open")
			}
			//Close main valve MExp
			MExp.SolenCmd("Close") // closes the valve
		}
	}
}
