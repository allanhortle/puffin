
var cmd = require('./cmd.js');

var install = function (arg) {
	cmd('brew', ['install', arg]);
}

var tap = function (arg) {
	cmd('brew', ['tap', arg]);
}

var brewInstall = function(env){
	cmd('brew', ['update'], true);
	cmd('brew', ['upgrade']);

	// taps
	tap('laurent22/massren');

	// installs
	install('massren');
	install('reattach-to-user-namespace');
	install('tmux');
	install('tig');
	install('git');
	install('fasd');
	install('node');
    install('fzf');
    install('ansiweather');

	cmd('brew', ['cleanup']);

	process.exit(0);
}

module.exports = brewInstall;
