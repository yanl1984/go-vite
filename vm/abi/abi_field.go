package abi

type abiField struct {
	Type    string
	Name    string
	Inputs  []Argument
	Outputs []Argument
}

func (field abiField) constructor() Method {
	return newMethod("", field.Inputs, nil)
}
func (field abiField) function() Method {
	return newMethod(field.Name, field.Inputs, nil)
}

func (field abiField) callback() Method {
	name := getCallBackName(field.Name)
	return newMethod(name, field.Inputs, nil)
}

func (field abiField) offChain() Method {
	return newMethod(field.Name, field.Inputs, field.Outputs)
}

func (field abiField) event() Event {
	indexed, nonIndexed := getEventInputs(field.Inputs)
	return Event{
		Name:             field.Name,
		Inputs:           field.Inputs,
		IndexedInputs:    indexed,
		NonIndexedInputs: nonIndexed,
	}
}

func (field abiField) variable() Variable {
	return Variable{
		Name:   field.Name,
		Inputs: field.Inputs,
	}
}
