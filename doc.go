// Package syndicate implements a beer club/syndicate system.
//
// Users buy beer and contribute it to the syndicate, and
// other syndicate users can checkout beer from those
// contributions.
//
// Total amounts contributed and checked out are tracked to
// verify that everyone is contributing their fair share.
//
// There is no authentication provided - this is designed for
// an honest and close knit group. This also allows users to
// contribute and checkout on behalf of others (i.e so only one
// user needs to update the system during a group meet).
package syndicate
