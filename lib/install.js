var cmd = require('./cmd.js');

module.exports = function brewInstall() {
    cmd('brew', ['update'], true);
    cmd('brew', ['upgrade']);

    // taps
    cmd('brew', ['tap', 'laurent22/massren']);

    // installs
    cmd('brew', ['install', 'fasd']);
    cmd('brew', ['install', 'git']);
    cmd('brew', ['install', 'htop']);
    cmd('brew', ['install', 'massren']);
    cmd('brew', ['install', 'node']);
    cmd('brew', ['install', 'reattach-to-user-namespace']);
    cmd('brew', ['install', 'tig']);
    cmd('brew', ['install', 'tmux']);
    cmd('brew', ['install', 'fzf']);
    cmd('brew', ['install', 'ripgrep']);
    cmd('brew', ['install', 'sops']);
    cmd('brew', ['install', 'vim']);
    cmd('brew', ['install', 'visidata']);
    cmd('brew', ['install', 'yarn']);
    cmd('brew', ['install', 'fnm']);

    cmd('brew', ['cleanup']);

    process.exit(0);
};
