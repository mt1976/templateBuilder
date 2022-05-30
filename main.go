package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/davecgh/go-spew/spew"
	_ "github.com/denisenkom/go-mssqldb"
	"github.com/google/uuid"

	"github.com/mt1976/templateBuilder/das"
	"github.com/mt1976/templateBuilder/logs"
	core "github.com/mt1976/templatebuiler/core"
)

type enrichments struct {
	ObjectName         string
	ObjectNameLower    string
	ObjectCamelCase    string
	ObjectGlyph        string
	ObjectTextClass    string
	EndpointRoot       string
	QueryString        string
	QueryField         string
	QueryFieldID       string
	SourceName         string
	Version            string
	Date               string
	Time               string
	Who                string
	Host               string
	FieldsList         []fields
	FriendlyName       string
	SQLTableName       string
	SQLSearchID        string
	SearchKey          string
	SourceType         string
	MessageList        []messages
	Path               string
	ProjectRepo        string
	UUID               string
	Title              string
	PageTitle          string
	UserMenu           string
	MenuHeader         string
	RangeUserMenuStart string
	RangeEnd           string
	MenuHREF           string
	MenuOnClick        string
	MenuGlyph          string
	MenuTextClass      string
	MenuText           string
	ItemsOnPageWc      string
	ItemList           string
	RangeItemList      string
	CanView            bool
	CanEdit            bool
	CanSave            bool
	CanNew             bool
	CanDelete          bool
	CanList            bool
	CanAPI             bool
	PropertiesName     string
	UsesAdaptor        bool
	CanExport          bool
	HasReverseLookup   bool
	ReverseLookup      string
	IsSpecial          bool
	OffersLookup       bool
	LookupID           string
	LookupName         string
	TemplateHeader     string
	TemplateFooter     string
	TemplateScripts    string
	TemplateAudit      string
	TitleText          string
}

type fields struct {
	FieldName     string
	Type          string
	Default       string
	FieldSQL      string
	Formatted     string
	TemplateField string
	Disabled      string
	Hidden        string
	ValueID       string
	IsMandatory   bool
	IsUserField   bool
	IsBaseField   bool
	IsLookup      bool
	IsOverride    bool
	IsExtra       bool
	LookupObject  string
	LookupField   string
	LookupValue   string
	RangeHTML     string
	WrapFieldName string
}

type messages struct {
	Message string
}

const (
	go_template   = ".go_template"
	html_template = ".html_template"
	json_template = ".json_template"
	nfo_template  = ".nfo_template"
	tableHeader   = "| %-35s | %-10s | %-10s | %-2s | %-2s | %-2s | %-2s | %-2s | %-15s | %-24s | %-24s |"
	tableRow      = "| %-35s | %-10s | %-10s | %-2s | %-2s | %-2s | %-2s | %-2s | %-15s | %-24s | %-24s |"
)

func main() {

	logs.Break()
	logs.Header("Template Generator")
	logs.Break()

	core.Initialise()

	displayApplicationHeader()

	logs.Break()
	logs.Header("Searching for work...")
	logs.Break()

	pwd, _ := os.Getwd()

	clItem := ""
	//log.Println(os.Args[1:], len(os.Args[1:]))
	//log.Println(os.Args[len(os.Args)-1], len(os.Args[len(os.Args)-1]))
	if len(os.Args[len(os.Args)-1]) > 1 {
		clItem = pwd + data_in() + "/" + os.Args[len(os.Args)-1]
	}
	//log.Println(clItem)

	var paths []string
	if clItem == "" {
		// Get list of files from a folder
		logs.Information("Searching...", data_in())
		paths = seekTableDefinitions(data_in())
		logs.Success("Found Files in " + data_in())
	} else {
		logs.Information("CL Specified", clItem+".cfg")
		paths = append(paths, clItem+".cfg")
	}

	noFiles := len(paths)

	logs.Information("Found ", fmt.Sprintf("%d %s %s", noFiles, " files in ", data_in()))

	// loop through files from Paths

	logs.Break()

	for i := 0; i < noFiles; i++ {
		// if last four character in paths[i] are ".cfg" then proceed otherwise skip this item
		fileExtension := paths[i][len(paths[i])-4:]
		//fmt.Println(fileExtension) // gives "lo"
		if fileExtension == ".cfg" {
			processTableDefinition(paths[i])
		}
	}
	logs.Break()
	logs.Success("Templating Complete")
	logs.Break()
}

//Get list of files from a folder
func seekTableDefinitions(dir string) []string {
	logs.Information("Searching...", "")
	dir = getPWD() + dir + "/"
	//logs.Information("In Queue Path", dir)
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}
	var paths []string
	for _, file := range files {
		if !file.IsDir() {
			paths = append(paths, filepath.Join(dir, file.Name()))
			logs.Information("Found File", file.Name())
		}
	}
	return paths
}

func processTableDefinition(configFile string) {
	logs.Processing(configFile)
	//	logs.Information("Populate", "Replacement Values")

	props := core.Config_Get(configFile)

	fmt.Printf("props: %v\n", props)

	e := setupEnrichment(props)

	csvPath := getPWD() + data_in() + "/" + e.ObjectName + ".csv"
	enriPath := getPWD() + data_in() + "/" + e.ObjectName + ".enri"
	logs.Information("CSV Path", csvPath)
	logs.Information("Enri Path", enriPath)

	if props["use"] == "db" {
		// Do nothing for now
		logs.Information("Getting List of fields from DB", props["server"]+" "+props["database"]+" "+props["tablename"])
		e = getFieldDefinitions_DB(e, props)
		e.SourceName = "APP"
	} else {
		if props["propertiesoverride"] == "special" {
			e.SourceName = "STATE"
		} else {
			e.SourceName = "SIENA"
		}
		logs.Information("Getting List of fields from CSV", csvPath)
		e = getFieldDefinitions_CSV(csvPath, e)
	}

	if strings.ToUpper(props["hasenrichments"]) == "Y" {
		//logs.Break()
		//	logs.Information("Getting Enrichment Fields from enri", csvPath)
		e = getEnrichmentFields_CSV(enriPath, e)
	}

	logs.Header("Generating Files")
	logs.Break()

	e = generateCodeArtifact("application", props, configFile, e)

	//e = generateCodeArtifact("adaptor", props, configFile, e)

	if e.CanAPI {
		e = generateCodeArtifact("api", props, configFile, e)
	}

	e = generateCodeArtifact("dao", props, configFile, e)

	e = generateCodeArtifact("datamodel", props, configFile, e)

	e = generateCodeArtifact("jobs", props, configFile, e)

	e = generateCodeArtifact("menu", props, configFile, e)

	e = generateHTMLArtifacts("html", props, configFile, e)

	e = generateCodeArtifact("catalog", props, configFile, e)

}

func generateCodeArtifact(a string, props map[string]string, configFile string, e enrichments) enrichments {
	if strings.ToUpper(props["create_"+a]) == "Y" || a == "catalog" {
		e = processCodeArtifact(a, configFile, a, e)
	} else {
		logs.Skipping(a)
	}
	return e
}

func processCodeArtifact(w string, p string, destFolder string, e enrichments) enrichments {
	logs.Processing(w)

	in_extn := ".go_template"
	out_extn := ".go_tmp"
	if core.Properties["deliverto"] != "" {
		out_extn = "_core.go"
	}

	if destFolder == "catalog" {
		destFolder = "design/catalog"
		out_extn = ".nfo"
		in_extn = ".nfo_template"
	}

	if destFolder == "api" {
		destFolder = "application"
		out_extn = "_api.go"
		in_extn = ".go_template"
	}

	if destFolder == "menu" {
		destFolder = "design/menu"
		out_extn = ".json"
		in_extn = ".json_template"
	}

	if destFolder == "html" {
		in_extn = ".html_template"
	}

	//if destFolder == "application" {
	//	out_extn = "_core" + out_extn
	//	}

	//spew.Dump(replacements)
	fp := e.Path + "/templates/" + w + in_extn

	t, err := template.ParseFiles(fp)
	if err != nil {
		logs.Error("Load Template :", err)
	}
	dest := "/" + destFolder + "/" + e.ObjectCamelCase + out_extn
	f, err := os.Create(data_out() + dest)
	if err != nil {
		logs.Error("Create file: ", err)
		return e
	}

	e.MessageList = append(e.MessageList, messages{Message: "* " + w + " (" + dest + ")"})

	err2 := t.Execute(f, e)
	if err2 != nil {
		logs.Error("Process Template", err2)
	}
	f.Close()
	logs.Created(f.Name())
	return e
}

func generateHTMLArtifacts(a string, props map[string]string, configFile string, e enrichments) enrichments {
	if strings.ToUpper(props["create_html"]) == "Y" {
		if e.CanList {
			e = generateHTMLArtifact("list", configFile, "html", e)
		} else {
			logs.Skipping("Listing is not enabled for this object")
		}

		if e.CanView {
			e = generateHTMLArtifact("view", configFile, "html", e)
		} else {
			logs.Skipping("Viewing is not enabled for this object")
		}

		if e.CanEdit {
			e = generateHTMLArtifact("edit", configFile, "html", e)
		} else {
			logs.Skipping("Editing is not enabled for this object")
		}

		if e.CanNew {
			e = generateHTMLArtifact("new", configFile, "html", e)
		} else {
			logs.Skipping("Creating is not enabled for this object")
		}

		if !e.CanExport {
			logs.Skipping("Exporting is not enabled for this object")
		}

	} else {
		logs.Skipping("html")
	}
	return e
}

func generateHTMLArtifact(w string, p string, destFolder string, e enrichments) enrichments {
	logs.Processing(w + html_template)

	userAction := strings.ToUpper(w[:1]) + w[1:]

	//spew.Dump(replacements)
	fp := e.Path + "/templates/" + w + html_template

	t, err := template.ParseFiles(fp)
	if err != nil {
		logs.Error("Load Template", err)
	}
	dest := "/" + destFolder + "/" + e.ObjectName + "_" + userAction + ".html"
	f, err := os.Create(data_out() + dest)
	if err != nil {
		logs.Error("Create file: ", err)
		return e
	}
	//spew.Dump(e)
	err2 := t.Execute(f, e)
	if err2 != nil {
		logs.Error("Process Template", err2)
	}
	f.Close()
	e.MessageList = append(e.MessageList, messages{Message: "* html -> " + userAction + " (" + dest + ")"})
	logs.Created(f.Name())
	return e
}

func getFieldDefinitions_CSV(filePath string, e enrichments) enrichments {
	// Load a csv file.
	//logs.Information("Read CSV", filePath)
	f, _ := os.Open(filePath)
	//logs.Information("File Open", filePath)
	// Create a new reader.
	r := csv.NewReader(f)
	//logs.Information("New Reader", filePath)
	displayTableHeader("Table")

	for {
		record, err := r.Read()

		//fmt.Printf("record: %v\n", record)

		// Stop at EOF.
		if err == io.EOF {
			break
		}

		if err != nil {
			logs.Fatal("Read CSV", err)
			panic(err)
		}

		colMand := false
		if record[3] == "true" {
			colMand = true
		}
		noInput := false
		if record[4] == "true" {
			noInput = true
		}
		if record[0] == "Name" && record[1] == "Type" {
			continue
		}
		e.FieldsList = addField(e, record[0], record[1], record[2], colMand, noInput)

	}
	logs.Break()
	return e
}

func getFieldDefinitions_DB(e enrichments, p map[string]string) enrichments {
	// Open Database Connection
	db, err := core.GlobalsDatabaseConnect(p)
	if err != nil {
		logs.Error("Database Connection", err)
	}
	//fmt.Printf("db: %v\n", db)

	//tsql := fmt.Sprintf("USE %s EXEC sp_columns '%s'", p["database"], p["sqltablename"])
	tsql := fmt.Sprintf("EXEC sp_columns '%s'", p["sqltablename"])
	//logs.Query(tsql)
	results, noFields, err := das.Query(db, tsql)
	//fmt.Printf("results: %v\n", results)
	//fmt.Printf("noFields: %v\n", noFields)
	//spew.Dump(results)
	if noFields == 0 {
		logs.Error("No Fields Found", err)
	}
	displayTableHeader("Table")
	for _, row := range results {
		colName := row["COLUMN_NAME"].(string)
		colType := row["TYPE_NAME"].(string)
		colMand := false
		if row["IS_NULLABLE"].(string) == "NO" {
			colMand = true
		}

		//logs.Message("found", fmt.Sprintf("name %v type %v nullable %v mandatory %t\n", colName, colType, row["IS_NULLABLE"], colMand))
		if colName == "ID" {
			colMand = true
		}
		colDefault := ""
		switch colType {
		case "varchar", "nvarchar", "char", "nchar", "text", "ntext":
			colDefault = ""
			colType = "String"
		case "int", "bigint", "smallint", "tinyint", "int64":
			colDefault = "0"
			colType = "Int"
		case "decimal", "numeric":
			colDefault = "0.00"
			colType = "Float"
		case "datetime", "smalldatetime", "date", "time", "datetime2", "datetimeoffset":
			colDefault = ""
			colType = "Time"
		case "float", "real", "money", "smallmoney":
			colDefault = "0.00"
			colType = "Float"
		case "int identity":
			colDefault = "0"
			colType = "Int"
		case "bigint identity":
			colDefault = "0"
			colType = "Int"
		case "bit":
			colDefault = "True"
			colType = "Bool"
		default:
			colType = "String"
			colDefault = ""
		}
		e.FieldsList = addField(e, colName, colType, colDefault, colMand, false)
	}
	return e
}

func displayApplicationHeader() {

	logs.Break()

	logs.Header("Application Information")
	logs.Break()

	logs.Header("Application")
	logs.Information("Name", core.Properties["appname"])
	logs.Information("Host Name", getHostName())

	logs.Information("Server Release", genReleaseName())
	logs.Information("Server Date", time.Now().Format(core.DATEFORMATUSER))

	logs.Information("Licence", core.Properties["licname"])
	logs.Information("Lic URL", core.Properties["liclink"])
	logs.Header("Runtime")
	logs.Information("GO Version", runtime.Version())
	logs.Information("Operating System", runtime.GOOS+" ("+runtime.GOARCH+")")

	logs.Default("Working Directory", getPWD())
	logs.Information("User", getUsername())
	logs.Header("Connectivity")
	logs.Default("Input", data_in())
	logs.Default("Output", data_out())
}

func displayTableHeader(in string) {
	logs.Break()
	logs.Header(in + " Information")
	logs.Break()
	logs.Information("Md ", "Mandatory")
	logs.Information("Cr ", "Core Fields")
	logs.Information("Ex ", "Extra Fields")
	logs.Information("Ov ", "Override of a Core Field")
	logs.Information("Lkp", "Lookup Field")
	logs.Break()
	info := fmt.Sprintf(tableHeader, "Field Name", "Type", "Default", "Md", "Cr", "Ex", "Ov", "L⬆", "⬆ Object", "⬆ Field", "⬇ Value")
	logs.Information(info, "")
	logs.Break()
}

func addField(en enrichments, fn string, tp string, df string, mand bool, noInput bool) []fields {

	en.FieldsList = addComplexField(en, fn, tp, df, mand, true, false, "", "", "", "", noInput, false, false)

	return en.FieldsList
}

func setupEnrichment(props map[string]string) enrichments {
	e := enrichments{ObjectName: props["objectname"]}
	//capitalize first character of enrichment.ObjectName
	logs.Information("Object Name", e.ObjectName)
	e.ObjectName = strings.Title(e.ObjectName)
	e.ObjectCamelCase = strings.ToLower(e.ObjectName[:1]) + e.ObjectName[1:]
	e.ObjectNameLower = strings.ToLower(e.ObjectName)
	e.Version = genReleaseName()
	e.Time = time.Now().Format(core.TIMEFORMATUSER)
	e.Date = time.Now().Format(core.DATEFORMATUSER)
	e.Host = getHostName()
	e.Who = getUsername()

	e.FriendlyName = props["friendlyname"]
	if e.FriendlyName == "" {
		e.FriendlyName = e.ObjectCamelCase
	}
	e.SQLTableName = props["sqltablename"]
	e.SQLSearchID = strings.TrimSpace(props["sqlsearchid"])
	e.QueryString = props["querystring"]
	e.QueryField = "{{." + props["queryfield"] + "}}"
	e.QueryFieldID = props["queryfield"]
	if props["endpointroot"] == "" {
		e.EndpointRoot = e.ObjectName
	} else {
		e.EndpointRoot = props["endpointroot"]
	}
	e.EndpointRoot = strings.ToUpper(e.EndpointRoot[:1]) + e.EndpointRoot[1:]

	e.Path = getPWD()
	e.ObjectGlyph = props["objectglyph"]
	e.ObjectTextClass = props["textclass"]
	e.ProjectRepo = props["projectrepo"] + "/"
	e.UUID = genUUID()

	e.PropertiesName = ""
	e.UsesAdaptor = false
	e.TemplateAudit = ""
	e.IsSpecial = false
	if props["propertiesoverride"] == "" {
		e.PropertiesName = "Application"
		e.TemplateAudit = wrapTemplate("audit")
	} else {
		e.UsesAdaptor = true
		if props["propertiesoverride"] == "special" {
			e.PropertiesName = "Application"
			e.IsSpecial = true
		} else {
			e.PropertiesName = props["propertiesoverride"]
		}
	}

	e = setupTemplateEnrichment(e, props)

	e = setupPermissions(e, props)

	e.HasReverseLookup = false
	e.ReverseLookup = ""
	if props["reverselookup"] != "" {
		e.HasReverseLookup = true
		e.ReverseLookup = props["reverselookup"]
	}

	// e.IsSpecial = false
	// if strings.ToUpper(props["isspecial"]) == "Y" {
	// 	e.IsSpecial = true
	// }

	e.OffersLookup = false
	if strings.ToUpper(props["offerslookup"]) == "Y" {
		e.OffersLookup = true
		e.LookupID = props["lookupid"]
		e.LookupName = props["lookupname"]
	}

	e.TemplateHeader = wrapTemplate("header")
	e.TemplateFooter = wrapTemplate("footer")
	e.TemplateScripts = wrapTemplate("scripts")
	spew.Dump(e)
	return e
}

func setupTemplateEnrichment(e enrichments, props map[string]string) enrichments {
	e.Title = wrap("Title")
	e.PageTitle = wrap("PageTitle")
	e.UserMenu = wrap("UserMenu")
	e.MenuHeader = "{{ (index .UserMenu 0).MenuHeaderText}}"
	e.RangeUserMenuStart = "{{range .UserMenu}}"
	e.RangeEnd = "{{end}}"
	e.MenuHREF = wrap("MenuHREF")
	e.MenuOnClick = wrap("MenuOnClick")
	e.MenuGlyph = wrap("MenuGlyph")
	e.MenuTextClass = wrap("MenuTextClass")
	e.MenuText = wrap("MenuText")
	e.ItemsOnPageWc = wrap("ItemsOnPage")
	e.ItemList = wrap("ItemList")
	e.RangeItemList = "{{range .ItemList}}"
	return e
}

func setupPermissions(e enrichments, props map[string]string) enrichments {
	e.CanView = true
	e.CanEdit = true
	e.CanDelete = true
	e.CanNew = true
	e.CanSave = true
	e.CanList = true
	e.CanExport = false
	e.CanAPI = false

	if strings.ToUpper(props["can_view"]) == "N" {
		e.CanView = false
	}
	if strings.ToUpper(props["can_edit"]) == "N" {
		e.CanEdit = false
	}
	if strings.ToUpper(props["can_delete"]) == "N" {
		e.CanDelete = false
	}
	if strings.ToUpper(props["can_new"]) == "N" {
		e.CanNew = false
	}
	if strings.ToUpper(props["can_save"]) == "N" {
		e.CanSave = false
		e.CanEdit = false
		e.CanNew = false
	}
	if strings.ToUpper(props["can_list"]) == "N" {
		e.CanList = false
	}

	if strings.ToUpper(props["can_export"]) == "Y" {
		e.CanExport = true
	}

	if strings.ToUpper(props["can_api"]) == "Y" {
		e.CanAPI = true
	}

	//spew.Dump(e)

	return e
}

func wrap(in string) string {
	return "{{." + in + "}}"
}

func wrapTemplate(in string) string {
	return "{{template " + enquote(in) + " .}}"
}

func enquote(in string) string {
	return "\"" + in + "\""
}

func getUsername() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return usr.Username
}

func genUUID() string {
	id := uuid.New()
	//fmt.Printf("github.com/google/uuid:         %s\n", id.String())
	return id.String()
}

func genReleaseName() string {
	return fmt.Sprintf("%s [r%s-%s]",
		core.Properties["releaseid"],
		core.Properties["releaselevel"],
		core.Properties["releasenumber"])
}

func getHostName() string {
	host, _ := os.Hostname()
	return host
}

func getPWD() string {
	thisPwd, _ := os.Getwd()
	return thisPwd
}

func data_out() string {
	do := ""
	if core.Properties["deliverto"] != "" {
		do = core.Properties["deliverto"]
	} else {
		do = getPWD() + core.Properties["data_out"]
	}
	return do
}

func data_in() string {
	return strings.TrimSpace(core.Properties["data_in"])
}

func getEnrichmentFields_CSV(filePath string, en enrichments) enrichments {

	//logs.Information("Read CSV", filePath)
	f, _ := os.Open(filePath)
	//logs.Information("File Open", filePath)
	// Create a new reader.
	r := csv.NewReader(f)
	//logs.Information("New Reader", filePath)
	//displayTableHeader("Enrichment")
	for {
		record, err := r.Read()
		//fmt.Printf("record: %v\n", record)
		// Stop at EOF.
		if err == io.EOF {
			break
		}

		if err != nil {
			logs.Fatal("Read Enri", err)
			panic(err)
		}
		if record[0] == "Type" && record[1] == "Field" {
			//logs.Information("Found", "Enrichments")
		} else {
			colMand := false
			if record[6] == "true" {
				colMand = true
			}

			isLookup := false
			isExtra := false
			isOverride := false
			lkObject := ""
			lkKeyField := ""
			lkValueField := ""
			lkRange := ""
			lkCodeField := ""

			//log.Println(record[0])
			suffix := "_Unknown"
			if record[0] == "Lookup" {
				suffix = "_Lookup"
				isLookup = true

				lkObject = record[2]
				lkKeyField = record[3]
				lkValueField = record[4]
				lkCodeField = record[8]
				lkRange = fmt.Sprintf("{{range .%s}}<option value=\"%s\">%s</option>{{end}}", record[1]+"_Lookup_List", wrap(lkCodeField), wrap(lkValueField))
			}

			if record[0] == "Extra" {
				isExtra = true
				suffix = "_Extra"
			}

			if record[0] == "Override" {
				isOverride = true
				suffix = ""
			}

			//log.Println(isLookup, isExtra, isOverride)

			noInput := true
			if record[5] == "true" {
				noInput = false
			}
			//fmt.Printf("record: %v\n", record)
			//fmt.Printf("record[5]: %v\n", record[5])
			//fmt.Printf("noInput: %v\n", noInput)
			//fmt.Printf("colMand: %v\n", colMand)

			en.FieldsList = addComplexField(en, record[1]+suffix, "String", record[7], colMand, false, isLookup, lkObject, lkKeyField, lkValueField, lkRange, noInput, isExtra, isOverride)
		}
	}
	logs.Break()
	return en
}

func addComplexField(en enrichments, fn string, tp string, df string, mand bool, baseField bool, isLookup bool, lkObject string, lkKeyField string, lkValueField string, lkRange string, noinp bool, isExtra bool, isOverride bool) []fields {

	// log parameters

	//log.Println("addComplexField:"+fn+" "+tp+" "+df+" "+strconv.FormatBool(mand)+" "+strconv.FormatBool(baseField)+" "+strconv.FormatBool(isLookup)+" "+lkObject+" "+lkKeyField+" "+lkValueField+" "+lkRange+" "+strconv.FormatBool(noinp), strconv.FormatBool(isExtra), strconv.FormatBool(isOverride))

	origfn := fn

	//if first charachter of fieldName is _ then replace _ with SYS

	noinput := ""
	hidden := ""
	userField := true

	if string(fn[0]) == "_" {
		//Convert fn to Title Case
		fn = strings.Replace(fn, "_", "", -1)
		fn = strings.ToUpper(fn[:1]) + fn[1:]
		//fmt.Println(fn)
		fn = "SYS" + fn
		noinput = "hidden"
		hidden = "hidden"
		userField = false
	}

	if noinp {
		noinput = "disabled"
	}
	fn = strings.ToUpper(fn[:1]) + fn[1:]

	info := fmt.Sprintf(tableRow, fn, tp, df, tf(mand), tf(baseField), tf(isExtra), tf(isOverride), tf(isLookup), lkObject, lkKeyField, lkValueField)
	tplField := "{{." + fn + "}}"
	en.FieldsList = append(en.FieldsList, fields{FieldName: fn,
		Type:          tp,
		Default:       df,
		FieldSQL:      origfn,
		Formatted:     info,
		TemplateField: tplField,
		Disabled:      noinput,
		Hidden:        hidden,
		ValueID:       wrap(fn),
		IsMandatory:   mand,
		IsUserField:   userField,
		IsBaseField:   baseField,
		IsLookup:      isLookup,
		LookupObject:  lkObject,
		LookupField:   lkKeyField,
		LookupValue:   lkValueField,
		RangeHTML:     lkRange,
		IsExtra:       isExtra,
		IsOverride:    isOverride})

	logs.Information(info, "")

	return en.FieldsList
}

func tf(in bool) string {
	if in {
		return "Y"
	}
	return ""
}
