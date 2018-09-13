# Quotable

There's this recurring thing where I write something that splits strings on spaces for a command parser or something, and then I end up wanting a way to give it a string with a space in it, and I reinvent the quoting wheel Yet Again, slightly differently and usually incompletely because I don't want to spend a lot of time on it.

So what if I had a good implementation and just used that?

This is not a replacement for `github.com/google/shlex`, which is significantly more powerful (and noticably more expensive, not that this should ever matter).

Usage:

```
results, err := quotable.Split(inputString, &quotable.Options{FancyBackslash: true, OnlySpaceIsSpace: false})
```

Results are the input string, split into substrings on whitespace. Double quotes (`"`) can escape spaces, and backslashes can escape spaces and quotes.

If `FancyBackslash` is set in the provided options, backslashes also support the usual Go backslash escapes, including `\x`, `\u`, and `\U`. If `OnlySpaceIsSpace` is set, tabs and other Unicode whitespace which aren't the actual space character are not treated as space. If the options parameter is nil, it's treated as all values being false.

Errors are reported, but even in the face of errors, `results` will contain a best-guess at what was probably intended. For instance:

```
results, err := quotable.Split(`a, "b`)
fmt.Printf("%#v\n%v\n", results, err)
// []string{"a", "b"}
// mismatched quote
```
