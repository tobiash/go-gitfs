# go-gitfs
 
This library currently provides a read-only filesystem abstraction (compatible with the [Afero](https://github.com/spf13/afero) `Fs` interface).
This allows reading from a particular reference of the repository without having a working directory. It also works on bare repositories.
It uses [Go-Git](https://github.com/src-d/go-git) to access the git repository.

Use `NewROFromHEAD` to get a filesystem view of the repository `HEAD` or `NewROFromRef` to get a filesystem representation of a particular *Ref*.

A writable `Fs` is work in progress - see `rw.go`. It currently leverages Afero's `CopyOnWriteFs` to overlay a writable filesystem on top of the read-only view - for example a `MemMapFs` could be used here. There is a `Commit` function to commit the contents of the filesystem to repository. However, `CopyOnWriteFs` has some limitations, for example it cannot really handle deletion of files properly.
