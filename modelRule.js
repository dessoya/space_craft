'use strict'

var util		= require('util')

function process(engine) {

    var tn = engine.GetVar('TableName')
    var v = engine.GetVar('UUIDFieldName')

    var type = engine.GetType('Fields')
	
	type.fields = [
		// { name: 'Field2CQL', type: 'map[string]string' },
		{ i:1, name: 'UUID',				type: 'gocql.UUID',	tag: '`cql:"' + v.value + '"`' },
		{ i:1, name: 'Exists',			type: 'bool' },
		{ i:1, name: 'IsLock',			type: 'bool',		tag: '`cql:"lock"`'},
		{ i:1, name: 'LockServerUUID',	type: 'gocql.UUID',	tag: '`cql:"lock_server_uuid"`' },
	].concat(type.fields)

	engine.replace(type, engine.CompileType(type))

	var a = '\
var Field2CQL = map[string]string{\n'

	for(var i = 0, c = type.fields, l = c.length; i < l; i++) {
		var f = c[i]
		if(!f.tag) continue
		var nameRE = /"(.+?)"/, ar = nameRE.exec(f.tag)
		if(ar) {
			var cqlname = ar[1]
			a += '	"' + f.name + '": "' + cqlname + '",\n'
		}
	}

	a += '}\n\
\n\
var InstallInfo = model.InstallInfo{ Init: Init }\n\
var LockServerUUID gocql.UUID\n\
var CQLSession *gocql.Session\n\
\n\
func Init(session *gocql.Session, serverUUID gocql.UUID) {\n\
	LockServerUUID = serverUUID\n\
	CQLSession = session\n\
}\n\
\n\
func Load(UUID gocql.UUID) (*Fields, error) {\n\
	m := &Fields{ UUID: UUID }\n\
	err := m.Load()\n\
	return m, err\n\
}\n\
\n\
func (m *Fields) Load() (error) {\n\
\n\
	m.Exists = false\n\
	var row = map[string]interface{}{}\n\
\n\
	if err := CQLSession.Query(`SELECT * FROM ' + tn.value + ' where ' + v.value + ' = ?`, m.UUID).MapScan(row); err != nil {\n\
		return err\n\
	}\n\
	m.Exists = true\n\
\n'
/*
	a += '\
		for key, value := range row {\n\
		switch key {\n'

	for(var i = 0, c = type.fields, l = c.length; i < l; i++) {
		var f = c[i]
		if(!f.tag) continue
		var nameRE = /"(.+?)"/, ar = nameRE.exec(f.tag)
		var cqlname = ar[1]
		a += '			case "' + cqlname + '":\n'
		switch(f.type) {
		case '*gocql.UUID':
		    a += '				v := value.(gocql.UUID)\n'
			a += '				m.' + f.name + ' = &v\n'

		break
		default:
			a += '				m.' + f.name + ' = value.(' + f.type + ')\n'
		}
	}

a += '\
		}\n\
	}\n'
	*/

	for(var i = 0, c = type.fields, l = c.length; i < l; i++) {
		var f = c[i]
		if(!f.tag) continue
		var nameRE = /"(.+?)"/, ar = nameRE.exec(f.tag)
		if(!ar) {
			continue
		}
		var cqlname = ar[1]
		/*
		var p = ''
		if(!f.i) {
			p 
		}
		*/

		switch(f.type) {
		case '*gocql.UUID':
		    a += '	v' + i + ' := row["' + cqlname + '"].(gocql.UUID)\n'
		    a += '	logger.String(fmt.Sprintf("' + f.name + ': %+v", v' + i + '))\n'
		    a += '	if v' + i + '.String() == "00000000-0000-0000-0000-000000000000" { m.' + f.name + ' = nil\n'
			a += '	} else { m.' + f.name + ' = &v' + i + '}\n'

		break
		default:
			a += '	m.' + f.name + ' = row["' + cqlname + '"].(' + f.type + ')\n'
		}
	}

	a += '\
\n\
	return nil\n\
}\n\
\n\
func Create() (*Fields, error) {\n\
    var err error\n\
	m := &Fields{ Exists: false }\n\
	for {\n\
		m.UUID, err = gocql.RandomUUID()\n\
		if err != nil {\n\
			return nil, err\n\
		}\n\
		var row = map[string]interface{}{}\n\
		var apply bool\n\
		if apply, err = CQLSession.Query(`insert into ' + tn.value + ' (' + v.value + ',create_time) values (?,now()) if not exists`, m.UUID).MapScanCAS(row); err != nil {\n\
			logger.Error(errors.New(err))\n\
			return nil, err\n\
		}\n\
		if apply {\n\
			break\n\
		}\n\
	}\n\
	return m, nil\n\
}\n\
\n\
var mutex sync.RWMutex\n\
var Models = map[string]*Fields{}\n\
\n\
func Access(uuid string) (*Fields) {\n\
\n\
    mutex.RLock()\n\
	m, ok := Models[uuid]\n\
    mutex.RUnlock()\n\
    if ok {\n\
    	return m\n\
    }\n\
    return nil\n\
}\n\
\n\
func Get(UUID gocql.UUID) (*Fields, error) {\n\
\n\
    var err error\n\
    uuid := UUID.String()\n\
    mutex.RLock()\n\
	m, ok := Models[uuid]\n\
    mutex.RUnlock()\n\
	if !ok {\n\
		m, err = Load(UUID)\n\
		if err != nil {\n\
			return nil, err\n\
		}\n\
		if m == nil {\n\
			return nil, nil\n\
		}\n\
	    mutex.Lock()\n\
		m2, ok := Models[uuid]\n\
	    if ok {\n\
		    mutex.Unlock()\n\
	    	return m2, nil\n\
	    }\n\
		Models[uuid] = m\n\
	    mutex.Unlock()\n\
	    m.Update(model.Fields{\n\
	    	"IsLock": true,\n\
	    	"LockServerUUID": LockServerUUID,\n\
	    })\n\
	}\n\
\n\
	return m, nil\n\
\n\
}\n'
	a += '\
\n\
func (m *Fields) Lock() (error) {\n\
\n\
    uuid := m.UUID.String()\n\
    mutex.Lock()\n\
	m, ok := Models[uuid]\n\
	if ok {\n\
	    mutex.Unlock()\n\
		return m.Load()\n\
	} else {\n\
		Models[uuid] = m\n\
	    mutex.Unlock()\n\
	    m.Update(model.Fields{\n\
	    	"IsLock": true,\n\
	    	"LockServerUUID": LockServerUUID,\n\
	    })\n\
	}\n\
\n\
	return nil\n\
\n\
}\n'
	a += '\
\n\
func (m *Fields) Unlock() error {\n\
\n\
    uuid := m.UUID.String()\n\
    mutex.Lock()\n\
	_, ok := Models[uuid]\n\
\n\
    if ok {\n\
    	delete(Models, uuid)\n\
    }\n\
    mutex.Unlock()\n\
\n\
    err := m.Update(model.Fields{\n\
    	"IsLock": nil,\n\
    	"LockServerUUID": nil,\n\
    })\n\
\n\
    return err\n\
}\n\
\n\
func (m *Fields) Update(fields model.Fields) error {\n\
\n\
	pairs := []string{}\n\
\n\
	for key, value := range fields {\n\
	    switch key {\n'

	for(var i = 0, c = type.fields, l = c.length; i < l; i++) {
		var f = c[i]
		if(!f.tag) continue

		// var nameRE = /"(.+?)"/, ar = nameRE.exec(f.tag)
		// var cqlname = ar[1]

		a += '		case "' + f.name + '":\n'
		switch(f.type) {
		case "*gocql.UUID":
			// a += '			switch t := value.(type) {\n'
			a += '			switch t := value.(type) {\n'
			a += '			case nil:\n'
			a += '			m.' + f.name + ' = nil\n'
			a += '			case gocql.UUID:\n'
			a += '			m.' + f.name + ' = &t\n'
			a += '			default:\n'
			a += '			m.' + f.name + ' = value.(' + f.type + ')\n'
			a += '			}\n'
			break

		case "gocql.UUID":
			// a += '			switch t := value.(type) {\n'
			a += '			switch value.(type) {\n'
			a += '			case nil:\n'
			a += '			m.' + f.name + ' = gocql.UUID{}\n'
			a += '			default:\n'
			a += '			m.' + f.name + ' = value.(' + f.type + ')\n'
			a += '			}\n'
			break

		case "string":
			// a += '			switch t := value.(type) {\n'
			a += '			switch value.(type) {\n'
			a += '			case nil:\n'
			a += '			m.' + f.name + ' = ""\n'
			a += '			default:\n'
			a += '			m.' + f.name + ' = value.(' + f.type + ')\n'
			a += '			}\n'
			break

		case "bool":
			// a += '			switch t := value.(type) {\n'
			a += '			switch value.(type) {\n'
			a += '			case nil:\n'
			a += '			m.' + f.name + ' = false\n'
			a += '			default:\n'
			a += '			m.' + f.name + ' = value.(' + f.type + ')\n'
			a += '			}\n'
			break

		case "int64":
			// a += '			switch t := value.(type) {\n'
			a += '			switch t := value.(type) {\n'
			a += '			case int:\n'
			a += '			m.' + f.name + ' = int64(t)\n'
			a += '			default:\n'
			a += '			m.' + f.name + ' = value.(' + f.type + ')\n'
			a += '			}\n'
			break

		case "float64":
			// a += '			switch t := value.(type) {\n'
			a += '			switch t := value.(type) {\n'
			a += '			case int:\n'
			a += '			m.' + f.name + ' = float64(t)\n'
			a += '			default:\n'
			a += '			m.' + f.name + ' = value.(' + f.type + ')\n'
			a += '			}\n'
			break

		default:
			a += '			m.' + f.name + ' = value.(' + f.type + ')\n'
		}
	}

		a += '\
		}\n\
		var pair = Field2CQL[key] + "="\n\
		switch t := value.(type) {\n\
		case nil:\n\
			pair += "null"\n\
		case bool:\n\
			pair += fmt.Sprintf("%v", t)\n\
		case int:\n\
			pair += fmt.Sprintf("%v", t)\n\
		case string:\n\
			pair += "\'" + t + "\'"\n\
		case float64:\n\
			pair += fmt.Sprintf("%v", t)\n\
		case int64:\n\
			pair += fmt.Sprintf("%v", t)\n\
		case *gocql.UUID:\n\
			pair += t.String()\n\
		case []*gocql.UUID:\n\
			a := []string{}\n\
			for _, uuid := range t {\n\
				a = append(a, uuid.String())\n\
			}\n\
			pair += "[" + strings.Join(a, ",") + "]"\n\
		case []gocql.UUID:\n\
			a := []string{}\n\
			for _, uuid := range t {\n\
				a = append(a, uuid.String())\n\
			}\n\
			pair += "[" + strings.Join(a, ",") + "]"\n\
		case []string:\n\
			a := []string{}\n\
			for _, s := range t {\n\
				a = append(a, `\'` + s + `\'`)\n\
			}\n\
			pair += "[" + strings.Join(a, ",") + "]"\n\
		case []int:\n\
			a := []string{}\n\
			for _, i := range t {\n\
				a = append(a, strconv.Itoa(i))\n\
			}\n\
			pair += "[" + strings.Join(a, ",") + "]"\n\
		case gocql.UUID:\n\
			pair += t.String()\n\
		default:\n\
			logger.Error(errors.New(fmt.Sprintf("unknown type: %+v",t)))\n\
		}\n\
		pairs = append(pairs, pair)\n\
	}\n\
	q := "update ' + tn.value + ' set " + strings.Join(pairs, ",") + " where ' + v.value + ' = " + m.UUID.String()\n\
	logger.String(q)\n\
\n\
	if err := CQLSession.Query(q).Exec(); err != nil {\n\
		logger.Error(errors.New(err))\n\
		return err\n\
	}\n\
\n\
	return nil\n\
\n\
}\n\
\n\
func GetLockedModels() ([]string) {\n\
	keys := []string{}\n\
    mutex.RLock()\n\
    for key, _ := range Models {\n\
    	keys = append(keys, key)\n\
    }\n\
    mutex.RUnlock()\n\
	return keys\n\
}\n\
'

	engine.append(a)


	return engine.content
}

module.exports = {
	process: process
}