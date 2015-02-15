var log = require('./logger.js');
var child_process = require('child_process');

var cmd = function (name, props, logCommand) {
	if(logCommand) {
		log.puffin(name, props.join(' '));
	}

	// make sure not to break
	try {
		var command = child_process.execFileSync(name, props, {stdio: "inherit"});
		if(command) {
			console.log(command);
		}
	} catch(e) {
	}

}

module.exports = cmd;
