
var cmd = require('./cmd.js');

var install = function (arg) {
	cmd('brew', ['install', arg]);
}

var tap = function (arg) {
	cmd('brew', ['tap', arg]);
}

var install = function(env){
	// cmd('brew', ['update']);
	cmd('brew', ['upgrade']);

	// taps
	tap('laurent22/massren');
	tap('bramstein/webfonttools');


	cmd('brew', ['install', 'massren']);
	cmd('brew', ['install', 'tmux']);
	cmd('brew', ['install', 'reattach-to-user-namespace']);
	cmd('brew', ['install', 'tig']);
	cmd('brew', ['install', 'sfnt2woff']);
	cmd('brew', ['install', 'sfnt2woff-zopfli']);
	cmd('brew', ['install', 'woff2']);
	cmd('brew', ['install', 'git']);
	cmd('brew', ['install', 'imagemagick', '--with-webp']);
	cmd('brew', ['install', 'node']);

	cmd('brew', ['cleanup']);

	process.exit(0);
}

module.exports = install;
