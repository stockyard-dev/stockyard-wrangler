package store
import ("database/sql";"fmt";"os";"path/filepath";"time";_ "modernc.org/sqlite")
type DB struct{db *sql.DB}
type Process struct{
	ID string `json:"id"`
	Name string `json:"name"`
	Command string `json:"command"`
	Status string `json:"status"`
	PID int `json:"pid"`
	AutoRestart string `json:"auto_restart"`
	LogPath string `json:"log_path"`
	CreatedAt string `json:"created_at"`
}
func Open(d string)(*DB,error){if err:=os.MkdirAll(d,0755);err!=nil{return nil,err};db,err:=sql.Open("sqlite",filepath.Join(d,"wrangler.db")+"?_journal_mode=WAL&_busy_timeout=5000");if err!=nil{return nil,err}
db.Exec(`CREATE TABLE IF NOT EXISTS processes(id TEXT PRIMARY KEY,name TEXT NOT NULL,command TEXT DEFAULT '',status TEXT DEFAULT 'stopped',pid INTEGER DEFAULT 0,auto_restart TEXT DEFAULT 'false',log_path TEXT DEFAULT '',created_at TEXT DEFAULT(datetime('now')))`)
return &DB{db:db},nil}
func(d *DB)Close()error{return d.db.Close()}
func genID()string{return fmt.Sprintf("%d",time.Now().UnixNano())}
func now()string{return time.Now().UTC().Format(time.RFC3339)}
func(d *DB)Create(e *Process)error{e.ID=genID();e.CreatedAt=now();_,err:=d.db.Exec(`INSERT INTO processes(id,name,command,status,pid,auto_restart,log_path,created_at)VALUES(?,?,?,?,?,?,?,?)`,e.ID,e.Name,e.Command,e.Status,e.PID,e.AutoRestart,e.LogPath,e.CreatedAt);return err}
func(d *DB)Get(id string)*Process{var e Process;if d.db.QueryRow(`SELECT id,name,command,status,pid,auto_restart,log_path,created_at FROM processes WHERE id=?`,id).Scan(&e.ID,&e.Name,&e.Command,&e.Status,&e.PID,&e.AutoRestart,&e.LogPath,&e.CreatedAt)!=nil{return nil};return &e}
func(d *DB)List()[]Process{rows,_:=d.db.Query(`SELECT id,name,command,status,pid,auto_restart,log_path,created_at FROM processes ORDER BY created_at DESC`);if rows==nil{return nil};defer rows.Close();var o []Process;for rows.Next(){var e Process;rows.Scan(&e.ID,&e.Name,&e.Command,&e.Status,&e.PID,&e.AutoRestart,&e.LogPath,&e.CreatedAt);o=append(o,e)};return o}
func(d *DB)Delete(id string)error{_,err:=d.db.Exec(`DELETE FROM processes WHERE id=?`,id);return err}
func(d *DB)Count()int{var n int;d.db.QueryRow(`SELECT COUNT(*) FROM processes`).Scan(&n);return n}
