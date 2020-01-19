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

	// installs
	install('fasd');
	install('git');
	install('htop');
	install('massren');
	install('node');
	install('reattach-to-user-namespace');
	install('tig');
	install('tmux');
    install('ansiweather');
    install('fzf');
    install('ripgrep');
    install('sops');
    install('todo-txt');
    install('vim');
    install('visidata');
    install('yarn');
    

	cmd('brew', ['cleanup']);
    
	process.exit(0);
}

module.exports = brewInstall;
