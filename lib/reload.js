var Rsync = require('rsync');
var path = require('path');

var cmd = require('./cmd.js');
var log = require('./logger.js');
var child_process = require('child_process');

var reload = function(env){
    const source = path.join(__dirname, '../files/');
    const destination = process.env.HOME;

    console.log(source, destination);
	var rsync = new Rsync.build({
		flags: 'avh',
		exclude: [
			'.DS_Store'
		],
		source,
		destination
	});

	var rsyncPid = rsync.execute(
		function callback(error, code, command) {
		    console.log(command);
			cmd('.', `${destination}/.zshrc`);
			log.success('dotfiles updated');
	    },
	    function stdoutHandler(data){},
	    function stderrHandler(error) {
			log.error(error);
	    	console.log(error);
	    }
	);
};

module.exports = reload;
