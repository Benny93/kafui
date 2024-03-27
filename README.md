# Kafui

A k9s inspired terminal ui for [kaf](https://github.com/birdayz/kaf)  
It uses the same configuration file as kaf so you can use your existing kaf configuration to browse between kafkas.

![asciicinema](asciicinema.gif)

![image topics](./doc/images/image.png)
![image groups](./doc/images/image-1.png)
![context select](./doc/images/image-2.png)


## Install

### Go install

1. **Set Environment Variables (For Unix-like Systems):**

   Make sure you have the `GOPATH` environment variable set. Add the following lines to your shell configuration file (e.g., `~/.bashrc` for Bash, `~/.zshrc` for Zsh):

   ```bash
   echo 'export GOPATH=$(go env GOPATH)' >> ~/.bashrc
   echo 'export PATH="$PATH:$GOPATH/bin"' >> ~/.bashrc
   ```

   For Bash, use `~/.bash_profile` instead of `~/.bashrc`.

   For Zsh, use `~/.zshrc`.

   These commands ensure that the `GOPATH` and `GOPATH/bin` are added to your `PATH` environment variable, allowing you to execute Go binaries globally.

2. **Set Environment Variables (For Windows):**

   Open Command Prompt as an administrator and run the following commands:

   ```cmd
   setx GOPATH "%USERPROFILE%\go"
   setx PATH "%PATH%;%GOPATH%\bin"
   ```

   These commands set the `GOPATH` environment variable to `%USERPROFILE%\go` and add `%GOPATH%\bin` to the `PATH` environment variable, respectively. After running these commands, you might need to restart your Command Prompt session for the changes to take effect.

3. **Install via Go:**

   Once the environment variables are set, you can install the package using `go install`. Run the following command:

   ```bash
   go install github.com/Benny93/kafui@latest
   ```

   This command fetches the latest version of the `kafui` package from the specified GitHub repository and installs it in your `GOPATH/bin` directory. After installation, you can execute the `kafui` command from anywhere in your terminal.

### Homebrew

If you're using Homebrew on macOS or Linux, you can easily install `kafui` using the following commands:

```bash
brew tap benny93/kafui
brew install kafui
```

This will tap into the `benny93/kafui` repository and install the `kafui` package on your system. 


### Downloader Script

Install via downloader script:

```bash
curl https://raw.githubusercontent.com/Benny93/kafui/main/godownloader.sh | BINDIR=$HOME/bin bash
```