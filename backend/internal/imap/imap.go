package imap

import (
    "github.com/emersion/go-imap"
    "github.com/emersion/go-imap/client"
)

func Delete(host, username, password string, uids []uint32) error {
    c, err := client.DialTLS(host, nil)
    if err != nil {
        return err
    }
    defer c.Logout()
    if err := c.Login(username, password); err != nil {
        return err
    }
    _, err = c.Select("INBOX", false)
    if err != nil {
        return err
    }
    seqset := new(imap.SeqSet)
    for _, u := range uids {
        seqset.AddNum(u)
    }
    if err := c.Store(seqset, "+FLAGS.SILENT", []interface{}{imap.DeletedFlag}, nil); err != nil {
        return err
    }
    return c.Expunge(nil)
}
