package store
import ("database/sql";"fmt";"os";"path/filepath";"time";_ "modernc.org/sqlite")
type DB struct{db *sql.DB}
type Worker struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	Command string `json:"command"`
	Instances int `json:"instances"`
	Status string `json:"status"`
	PID int `json:"pid"`
	RestartCount int `json:"restart_count"`
	LastStartAt string `json:"last_start_at"`
	CreatedAt string `json:"created_at"`
}
func Open(d string)(*DB,error){if err:=os.MkdirAll(d,0755);err!=nil{return nil,err};db,err:=sql.Open("sqlite",filepath.Join(d,"wrangler.db")+"?_journal_mode=WAL&_busy_timeout=5000");if err!=nil{return nil,err}
db.Exec(`CREATE TABLE IF NOT EXISTS workers(id TEXT PRIMARY KEY,name TEXT NOT NULL,type TEXT DEFAULT 'process',command TEXT DEFAULT '',instances INTEGER DEFAULT 1,status TEXT DEFAULT 'stopped',pid INTEGER DEFAULT 0,restart_count INTEGER DEFAULT 0,last_start_at TEXT DEFAULT '',created_at TEXT DEFAULT(datetime('now')))`)
return &DB{db:db},nil}
func(d *DB)Close()error{return d.db.Close()}
func genID()string{return fmt.Sprintf("%d",time.Now().UnixNano())}
func now()string{return time.Now().UTC().Format(time.RFC3339)}
func(d *DB)Create(e *Worker)error{e.ID=genID();e.CreatedAt=now();_,err:=d.db.Exec(`INSERT INTO workers(id,name,type,command,instances,status,pid,restart_count,last_start_at,created_at)VALUES(?,?,?,?,?,?,?,?,?,?)`,e.ID,e.Name,e.Type,e.Command,e.Instances,e.Status,e.PID,e.RestartCount,e.LastStartAt,e.CreatedAt);return err}
func(d *DB)Get(id string)*Worker{var e Worker;if d.db.QueryRow(`SELECT id,name,type,command,instances,status,pid,restart_count,last_start_at,created_at FROM workers WHERE id=?`,id).Scan(&e.ID,&e.Name,&e.Type,&e.Command,&e.Instances,&e.Status,&e.PID,&e.RestartCount,&e.LastStartAt,&e.CreatedAt)!=nil{return nil};return &e}
func(d *DB)List()[]Worker{rows,_:=d.db.Query(`SELECT id,name,type,command,instances,status,pid,restart_count,last_start_at,created_at FROM workers ORDER BY created_at DESC`);if rows==nil{return nil};defer rows.Close();var o []Worker;for rows.Next(){var e Worker;rows.Scan(&e.ID,&e.Name,&e.Type,&e.Command,&e.Instances,&e.Status,&e.PID,&e.RestartCount,&e.LastStartAt,&e.CreatedAt);o=append(o,e)};return o}
func(d *DB)Update(e *Worker)error{_,err:=d.db.Exec(`UPDATE workers SET name=?,type=?,command=?,instances=?,status=?,pid=?,restart_count=?,last_start_at=? WHERE id=?`,e.Name,e.Type,e.Command,e.Instances,e.Status,e.PID,e.RestartCount,e.LastStartAt,e.ID);return err}
func(d *DB)Delete(id string)error{_,err:=d.db.Exec(`DELETE FROM workers WHERE id=?`,id);return err}
func(d *DB)Count()int{var n int;d.db.QueryRow(`SELECT COUNT(*) FROM workers`).Scan(&n);return n}

func(d *DB)Search(q string, filters map[string]string)[]Worker{
    where:="1=1"
    args:=[]any{}
    if q!=""{
        where+=" AND (name LIKE ?)"
        args=append(args,"%"+q+"%");
    }
    if v,ok:=filters["type"];ok&&v!=""{where+=" AND type=?";args=append(args,v)}
    if v,ok:=filters["status"];ok&&v!=""{where+=" AND status=?";args=append(args,v)}
    rows,_:=d.db.Query(`SELECT id,name,type,command,instances,status,pid,restart_count,last_start_at,created_at FROM workers WHERE `+where+` ORDER BY created_at DESC`,args...)
    if rows==nil{return nil};defer rows.Close()
    var o []Worker;for rows.Next(){var e Worker;rows.Scan(&e.ID,&e.Name,&e.Type,&e.Command,&e.Instances,&e.Status,&e.PID,&e.RestartCount,&e.LastStartAt,&e.CreatedAt);o=append(o,e)};return o
}

func(d *DB)Stats()map[string]any{
    m:=map[string]any{"total":d.Count()}
    rows,_:=d.db.Query(`SELECT status,COUNT(*) FROM workers GROUP BY status`)
    if rows!=nil{defer rows.Close();by:=map[string]int{};for rows.Next(){var s string;var c int;rows.Scan(&s,&c);by[s]=c};m["by_status"]=by}
    return m
}
