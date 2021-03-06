/*
Copyright (c) 2016 Nick Potts
Licensed to You under the GNU GPLv3
See the LICENSE file at github.com/npotts/homehub/LICENSE

This file is part of the HomeHub project
*/

package homehub

/*An Attendant performs the function of listening for data messages and forwarding them to a backend to store*/
type Attendant interface {
	Use(Backend) //where do we aim messages
	Stop()       //cease operations and exit
}

/*A Backend supports multiple attendants that gather
messages by various means*/
type Backend interface {
	Register(Datam) error //registers a datam
	Store(Datam) error    //Stores datam
	Stop()                //cease operations
}

/*GoodSample is a sample of a good Datam*/
var GoodSample = Datam{
	Table: "test",
	Data: map[Alphabetic]Field{
		Alphabetic("float"):  Field{Value: 1.0, mode: fmFloat},
		Alphabetic("string"): Field{Value: "str", mode: fmString},
		Alphabetic("int"):    Field{Value: 1, mode: fmInt},
		Alphabetic("bool"):   Field{Value: false, mode: fmBool},
	},
}
