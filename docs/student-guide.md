# Install Clio, build a site, publish it

Do these parts in order. Commands under **Terminal** run in the shell. Commands under **Neovim** run inside the editor. Quit Neovim before the next **Terminal** block.

You need a phone with Termux or a Linux computer, a [GitHub](https://github.com) account, a [DeepSeek API key](https://platform.deepseek.com/api_keys), and later a server login (hostname, username, and a key or password). Git, GitHub, and SSH are shell commands.

On a phone, install Termux from [F-Droid](https://f-droid.org/en/packages/com.termux/) or the [Termux releases](https://github.com/termux/termux-app/releases). The Google Play copy is too old. You should see a `$` prompt. On a computer, open the terminal.

## 1. Install Clio

**Terminal**

```bash
curl -fsSL https://clipilot.themobileprof.com/clio | sh
```

If that fails:

```bash
curl -sfL https://raw.githubusercontent.com/themobileprof/clio/main/install.sh | bash
```

Close the terminal and open it again. If `clio` is still missing on a computer:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

**Check:** `clio` shows `>>`. Type `exit`.

## 2. Download the wizards

**Terminal**

```bash
clio
```

At `>>` type `sync`, wait until it finishes, then `exit`.

On Termux, run wizards with `clio-run-module` in the shell. Running one at `>>` can crash the app.

**Check:** `clio`, then `setup`, lists Neovim and Git. Type `exit`.

## 3. Prepare the device

**Terminal, phone only.** Skip this on a computer.

```bash
clio-run-module termux_setup setup
```

Say yes to storage and the shell. When it asks **Build HTML pages?**, say yes. Pico CSS is saved at `~/.clio/share/pico.classless.min.css`. Say no to extra languages unless you want them. Plugins and the API key are next. If your shell switched to Zsh, open a new Termux session.

## 4. Set up Neovim

**Terminal**

```bash
clio-run-module neovim_setup setup
```

Paste your DeepSeek key when asked. Input is hidden. It is stored in `~/.config/gp/deepseek.key`, outside the site, so it is not pushed to GitHub.

If plugin install fails:

```bash
nvim
```

**Neovim:** `:PlugInstall`, wait, then `:qa`.

**Terminal, after you quit**

```bash
nvim --version
```

| Keys | Action |
|------|--------|
| `,ga` | Generate after the cursor |
| `,gr` | Rewrite the selection |
| `:w` | Save |
| `:q` | Quit |

Press `i` to type, Esc to return to normal mode. The first reply can take a while. The text in the file is the answer.

**If generate says the key is missing:** `nvim ~/.config/gp/deepseek.key`, press `i`, paste the key, Esc, `:w`, `:q`.

## 5. Set up Git and GitHub

**Terminal**

```bash
clio-run-module git_setup setup
gh auth login
```

Use your GitHub email in the wizard. For `gh auth login`, choose GitHub.com, HTTPS, and a web browser.

**Check:**

```bash
gh auth status
git config --global user.name
```

## 6. Generate the site with Pico CSS

[Pico CSS](https://picocss.com) classless 2.1.1 styles ordinary tags (`header`, `main`, `nav`, headings, links). The model writes HTML only.

**Terminal, before Neovim**

```bash
mkdir -p ~/my-site/css
cd ~/my-site
cp ~/.clio/share/pico.classless.min.css css/
nvim index.html
```

If that file is missing, download it once and copy it again:

```bash
mkdir -p ~/.clio/share
curl -fsSL -o ~/.clio/share/pico.classless.min.css \
  https://cdn.jsdelivr.net/npm/@picocss/pico@2.1.1/css/pico.classless.min.css
cp ~/.clio/share/pico.classless.min.css css/
```

**Neovim:** press `,ga` and type the page. On an HTML file the editor adds the Pico rules to that prompt.

```text
A personal homepage with my name, a short intro, and a link to about.html.
```

`:w` then `:q`.

**Terminal, after you quit**

```bash
nvim about.html
```

**Neovim:** press `,ga` and type:

```text
An about page with one paragraph and a link back to index.html.
```

`:w` then `:q`.

**Terminal, after you quit**

```bash
find . -type f | sort
grep pico index.html
```

You should see `css/pico.classless.min.css`, `index.html`, and `about.html`.

Preview on a computer by opening `index.html`. On a phone:

```bash
python3 -m http.server 8080
```

If `python3` is missing: `pkg install -y python`, then run it again. Open `http://127.0.0.1:8080`. Press Ctrl-C when you are done.

**If the page is empty:** fix the key in part 4 and run `,ga` again. **If there is no layout:** `ls -l css` should show a non-empty Pico file. Copy `~/.clio/share/pico.classless.min.css` again if it does not.

## 7. Push to GitHub

**Terminal.** Stay in `~/my-site`. Neovim must be closed.

```bash
git init
git add index.html about.html css
git status
git commit -m "Add my first homepage"
gh repo create my-site --public --source=. --remote=origin --push
```

If the name `my-site` is taken, pick another and run `gh repo create` again.

**Check:** `gh repo view` prints a URL. Open it. The HTML files and `css` are there.

## 8. Deploy over SSH

You need a hostname and username from your teacher or host. This guide does not create the server.

**Terminal, on your device**

```bash
pkg install -y openssh
```

Skip that install on a computer if `ssh-keygen` already exists.

```bash
ssh-keygen -t ed25519 -C "student-site"
cat ~/.ssh/id_ed25519.pub
```

Press Enter for the file location and an empty passphrase. Send the `.pub` line to whoever runs the server. Keep `~/.ssh/id_ed25519` on your device.

```bash
ssh USER@HOST
```

Type `yes` at the first fingerprint prompt.

**Check:** the prompt is the server. `exit` returns you to your device.

**Terminal, on the server, after `ssh`**

With `sudo`, this serves the site on port 80:

```bash
sudo apt-get update
sudo apt-get install -y nginx
sudo mkdir -p /var/www/my-site
sudo chown "$USER" /var/www/my-site
git clone https://github.com/YOUR_USER/my-site.git /var/www/my-site
sudo tee /etc/nginx/sites-available/my-site >/dev/null << 'EOF'
server {
    listen 80;
    server_name _;
    root /var/www/my-site;
    index index.html;
}
EOF
sudo ln -sf /etc/nginx/sites-available/my-site /etc/nginx/sites-enabled/my-site
sudo rm -f /etc/nginx/sites-enabled/default
sudo nginx -t
sudo systemctl reload nginx
```

Open `http://HOST` in a browser.

Without `sudo`:

```bash
mkdir -p ~/sites
git clone https://github.com/YOUR_USER/my-site.git ~/sites/my-site
cd ~/sites/my-site
python3 -m http.server 8080
```

Open `http://HOST:8080`. This stops when you press Ctrl-C or the SSH session ends.

## 9. Publish a change

**Terminal, before Neovim**

```bash
cd ~/my-site
nvim index.html
```

**Neovim:** edit with `i` or `,gr`. `:w` then `:q`.

**Terminal, after you quit**

```bash
git add .
git commit -m "Update the homepage"
git push
```

**Terminal, on the server**

```bash
ssh USER@HOST
cd /var/www/my-site
git pull
```

If you used the home-folder copy: `cd ~/sites/my-site` before `git pull`. Reload the browser.

## When you forget a command

**Terminal**

```bash
clio
```

Ask in a sentence (`how do I see my git status`), then `exit`. Run Git, `gh`, and `ssh` yourself.

## Done when

- [ ] `clio` starts
- [ ] `,ga` writes the HTML page and `,gr` changes a selection
- [ ] `gh repo view` opens the GitHub project
- [ ] `ssh USER@HOST` logs you in
- [ ] the browser shows the site, and a later `git push` plus `git pull` changes it
