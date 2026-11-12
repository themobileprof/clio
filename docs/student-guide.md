# Install Clio, build a site, publish it

Do these parts in order. Commands under **Terminal** run in the shell. Commands under **Neovim** run inside the editor. Quit Neovim before the next **Terminal** block.

You need a phone with Termux or a Linux computer, a [GitHub](https://github.com) account, a [DeepSeek API key](https://platform.deepseek.com/api_keys), and later a server login (hostname, username, and a key or password). Git, GitHub, and SSH are shell commands.

On a phone, open [clio.themobileprof.com/termux](https://clio.themobileprof.com/termux) and tap Download Termux. That page gives you the GitHub file for this phone. Android shows a warning. Open the details and tap Install anyway. The Play Store copy cannot run a program you download. You should see a `$` prompt.

On Windows, open [clio.themobileprof.com/windows](https://clio.themobileprof.com/windows) and finish that page first. You will get an Ubuntu window. The commands below go in that window, not in PowerShell. On Linux, open the terminal.

## 1. Install Clio

**Terminal**

```bash
curl -fsSL https://clio.themobileprof.com/clio | sh
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

On Termux, run each wizard in the shell, after you type `exit`. Running one at `>>` can crash the app.

**Check:** Run `clio` again. At the `>>` prompt, type `setup`. You should see a list that names Neovim and Git, each marked downloaded or not downloaded. Type `exit` to leave Clio.

## 3. Prepare the device

**Terminal, phone only.** Skip this on a computer.

```bash
clio termux_setup
```

Say yes to storage and the shell. When it asks **Build HTML pages?**, say yes. Pico CSS is saved at `~/.clio/share/pico.min.css`. Say no to extra languages unless you want them. Plugins and the API key are next. If your shell switched to Zsh, open a new Termux session.

## 4. Set up Neovim

**Terminal**

```bash
clio neovim_setup
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
clio git_setup
gh auth login
```

Use your GitHub email in the wizard. For `gh auth login`, choose GitHub.com, HTTPS, and a web browser.

**Check:**

```bash
gh auth status
git config --global user.name
```

## 6. Generate the site

The site wizard writes the folder. You choose a Pico color by name. Light and dark are already on the page, with System, Light, and Dark in the header. The wizard does not write a stylesheet.

**Terminal**

```bash
clio site_setup
```

Answer the folder, the site name, the pages, one sentence for each page, and a Pico color. Press Enter on the color to keep azure. Other names are amber, blue, cyan, fuchsia, green, grey, indigo, jade, lime, orange, pink, pumpkin, purple, red, sand, slate, violet, yellow, and zinc.

Settings are in `site.config.js` in that folder. The comments list the other colors and the appearance choices: `system`, `light`, and `dark`. After you change the color, run `clio site_setup` again and say yes. That downloads the matching Pico file and keeps the pages you already wrote.

Photos go in `images/` before you edit a page. Name each file for what it shows, such as `images/hero.jpg`.

**Preview**

```bash
cd ~/my-site
python3 -m http.server 8080
```

If `python3` is missing: `pkg install -y python`, then run it again. Open `http://127.0.0.1:8080`. Tap System, Light, and Dark. Press Ctrl-C when you are done.

**Neovim, to change a page**

```bash
nvim index.html
```

Move into one `clio-view` block. Select those lines with `V`, press `,gr`, and say what to change. On this file the editor keeps the header, the Pico theme, and the light and dark buttons.

**If a page is empty:** fix the key in part 4, then select that view and run `,gr` again. **If there is no layout:** `ls css` should show a non-empty Pico file. Run `clio site_setup` again and say yes.

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
