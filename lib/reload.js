var Rsync = require('rsync');

var cmd = require('./cmd.js');
var log = require('./logger.js');
var child_process = require('child_process');

var reload = function(env){
	var rsync = new Rsync.build({
		flags: 'avh',
		exclude: [
			'.git/',
			'bin/',
			'lib/',
			'.gitignore',
			'.DS_Store',
			'package.json',
			'bin/',
			'README.md'
		],
		source: '.',
		destination: '~'
	});

	var rsyncPid = rsync.execute(
		function callback(error, code, command) {
			cmd(process.env.HOME + '/.zshrc');
			cmd('echo', ['$(SHELL)'])
			log.success('dotfiles updated');
	    },
	    function stdoutHandler(data){
	        // log.puffin(data);
	    },
	    function stderrHandler(error) {
	        log.error(error);
	    }
	);
}

module.exports = reload;
