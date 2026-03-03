/*
Package middleware provides a Handler/Middleware pattern for intercepting and
transforming requests and responses in the Lamar SDK.

The middleware pattern allows you to:

  - Add logging, metrics, and tracing
  - Implement retry logic
  - Add request/response transformation
  - Handle errors gracefully

# Handler Interface

The Handler interface represents a request handler:

	type Handler interface {
	    Handle(ctx context.Context, req Request) (Response, error)
	}

# Middleware Type

A Middleware wraps a Handler to add behavior:

	type Middleware func(Handler) Handler

# Creating Middleware

	middleware := func(next Handler) Handler {
	    return HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
	        // Pre-processing
	        resp, err := next.Handle(ctx, req)
	        // Post-processing
	        return resp, err
	    })
	}

# Chaining Middleware

	handler := middleware.Chain(
	    Logging(logger),
	    Metrics(collector),
	    Recover(),
	)(baseHandler)

# Built-in Middleware

  - Logging: Logs requests and responses
  - Metrics: Records metrics for requests
  - Recover: Recovers from panics and converts to errors
*/
package middleware
