
var cmd = require('./cmd.js');

var cask = function (arg) {
	cmd('brew', ['cask', 'install', arg]);
}

var tap = function (arg) {
	cmd('brew', ['tap', arg]);
}


var brewInstallApps = function(env){
    
    cmd('brew', ['update'], true);

	// installs
    cask('alfred');
    cask('bettertouchtool');
    cask('dropbox');
    cask('1password');
    cask('google-chrome');
    cask('firefox');
    cask('iterm2');
    cask('slack');
    cask('spotify');
    cask('the-unarchiver');
    cask('transmit');
    cask('visual-studio-code');
    
	process.exit(0);
}

module.exports = brewInstallApps;
