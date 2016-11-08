
var cmd = require('./cmd.js');

var cask = function (arg) {
	cmd('brew', ['cask', 'install', arg]);
}

var tap = function (arg) {
	cmd('brew', ['tap', arg]);
}


var brewInstallApps = function(env){
    
    cmd('brew', ['update'], true);

    // taps
    tap('caskroom/cask');

	// installs
    cask('alfred');
    cask('bettertouchtool');
    cask('google-chrome');
    cask('firefox');
    cask('iterm2');
    cask('slack');
    cask('spotify');
    cask('sublime-text');
    cask('the-unarchiver');
    cask('transmit');
    
	process.exit(0);
}

module.exports = brewInstallApps;
