var cmd = require('./cmd.js');

function install(arg) {
	cmd('brew', ['install', arg]);
}

function tap(arg) {
	cmd('brew', ['tap', arg]);
}

function gitClone(arg) {
    cmd('git',['clone', arg]);
}

function brewInstall(env){
	cmd('brew', ['update'], true);
	cmd('brew', ['upgrade']);

	// taps
	tap('laurent22/massren');
	tap('koekeishiya/formulae');

	// installs
	install('massren');
	install('reattach-to-user-namespace');
	install('tmux');
	install('tig');
	install('git');
	install('hub');
	install('fasd');
	install('node');
    install('fzf');
    install('ripgrep');
    install('ansiweather');
    install('skhd');
    install('chunkwm');

	cmd('brew', ['cleanup']);
    
	process.exit(0);
}

module.exports = brewInstall;
