# is
Lightweight testing mini-framework for Go.

* Simple, easy to write and read
* is.Equal, is.OK and is.NoErr
* Use comments to add descriptions which show up when tests fail

Usage

The following code shows a range of useful ways you can use
the helper methods:

```go
func Test(t *testing.T) {

	is := is.New(t)
	
	signedin, err := isSignedIn(ctx)
	is.NoErr(err)            // isSignedIn error
	is.Equal(signedin, true) // must be signed in
	
	body := readBody(r)
	is.OK(strings.Contains(body, "Hi there"))
	
}
```
