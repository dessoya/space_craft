'use strict'

var fs			= require('fs')

// console.log(process.argv)

var rule = require(__dirname + '\\' + process.argv[2])
var sourceFileName = process.argv[3]
var destFileName = process.argv[4]

var content = '' + fs.readFileSync(sourceFileName)

String.prototype.repeat = function(count) {
	var t = ''
	while(count -- && count > 0) {
		t += this
	}
	return t
}

var engine = {
	
	content: content,

	GetVar: function(varName) {
		var varRE = new RegExp('var\\s+' + varName + '\\s+=\\s+"(.+?)"', 'm')
		// console.log(varRE)
		var a = varRE.exec(this.content)
		// console.log(a)
		if(!a) {
			return null
		}
		return { value: a[1] }		
	},

	GetType: function(typeName) {

		var startRE = new RegExp('type\\s+' + typeName + '\\s+struct\\s+\\{', 'm')

		var a = startRE.exec(this.content)
		if(!a) {
			return null
		}

		var info = { type: typeName, start: a.index, fields: [ ] }

		var fieldRE = /([a-zA-Z\d_]+)\s+(\S+)\s*(`\S+`)?|(\})/gm
		var from = a.index + a[0].length
		fieldRE.lastIndex = from

		do {
			a = fieldRE.exec(this.content)
			if(!a || a[4]) break
			info.fields.push({ name: a[1], type: a[2], tag: a[3] })
		}
		while(true)

		info.end = fieldRE.lastIndex

		// info.e = this.content.substr(info.end, 20)
		
		return info
	},

	replace: function(placeInfo, text) {
		this.content = this.content.substr(0, placeInfo.start) + text + this.content.substr(placeInfo.end)
	},

	CompileType: function(type) {
		var text = 'type ' + type.type + ' struct {\n'

		var nl = 0, tl = 0 // , taglen = 0

		for(var i = 0, c = type.fields, l = c.length; i < l; i ++) {
		    var f = c[i]
		    if(f.name.length > nl) nl = f.name.length
		    if(f.type.length > tl) tl = f.type.length
		    // if(f.tag && f.tag.length > tagl) tagl = f.tag.length
		}

		var tabSize = 4

		nl = Math.floor(nl / tabSize) + 2
		tl = Math.floor(tl / tabSize) + 2

		for(var i = 0, c = type.fields, l = c.length; i < l; i ++) {
		    var f = c[i]

			text += '\t' + f.name + '\t'.repeat(nl - Math.floor(f.name.length / tabSize)) + f.type
			if(f.tag) {
				text += '\t'.repeat(tl - Math.floor(f.type.length / tabSize)) + f.tag
			}
			text += '\n'
		}
		text += '}\n'
		return text
	},

	append: function(text) {
		this.content += text
	}
}

var result = rule.process(engine, content)



fs.writeFileSync(destFileName, result)