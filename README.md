# ssh-git-go

## What is it?

A really small and silly SSH server that allows anonymous access to git repositories. This means you can allow cloning of a repository over SSH without the user having a valid account on the server. 

## Why is this useful?

Generally there shouldn't be a reason to do this, since the http:// handler already makes it easy to give anonymous access to repositories. In some instances, such as an exploit in git, it might be useful to allow anonymous access where a user isn't prompted for credentials etc.

The downside of SSH is that host key verification is enforced, meaning it still requires some interaction by the user to accept the host key. But once that host key is accepted, it is smooth sailing. 

## How to use

Grab the code:

```bash
go get github.com/staaldraad/ssh-git-go
```

Then use:

```bash
./ssh-git-go -h
Usage of ssh-git-go:
  -d string
        The directory where the git repositories are (default "./repos")
  -i string
        The interface to listen on (default "0.0.0.0")
  -p int
        Port to use (default 2221)
  -s string
        Where to find the host-key (default "./id_rsa")
```

The usage options should be pretty self-explanitory. Before running you'll need to generate a host-key for SSH to use, to do this use `ssh-keygen`:

```bash
ssh-keygen -t rsa
```

The `-d` parameter allows specifying the location where your git repositories live (these have to be bare repositories).

### Example

Run the server:

```bash
./ssh-git-go -p 2221 -d /tmp/repos
2018/10/18 16:42:32 New listener started on 0.0.0.0:2221
2018/10/18 16:42:32 Serving repositories found in /tmp/repos
```

Now a normal `git clone` should work against the server. Lets assume there is a folder called `meh.git` in the `/tmp/repos` directory and this has the bare repository.

On the client:

```bash
git clone ssh://serveraddress:2221/meh.git
Cloning into 'meh'...
remote: Counting objects: 3, done. 
Remote: Total 3 (delta 0), reused 0 (delta 0) 
Receiving objects: 100% (3/3), done.
```

On the server you should see:

```bash
2018/10/18 16:46:53 New SSH connection from [::1]:60864 (SSH-2.0-OpenSSH_7.4p1 Debian-10+deb9u4)
2018/10/18 16:46:53 Requesting repo: /tmp/repos/meh.git
```

# License

Made with tears by @staaldraad and distributed under [MIT](https://github.com/staaldraad/ssh-git-go/blob/master/LICENSE) license. 

Kudos and hate on Twitter: [@_staaldraad](https://twitter.com/_staaldraad)