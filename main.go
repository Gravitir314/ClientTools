package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

const (
	menu = "1. Check for updates.\n" +
		"2. Download client.\n" +
		"3. Export resources.\n" +
		"4. Add proxy server. (PlayerGlobal required)\n" +
		"> "
)

var (
	version     string
	workingPath string
)

func checkErr(err error) {
	if err != nil {
		log.Println(err)
		checkMenu()
	}
}

func checkUpdates() {
	resp, err := http.Get("http://www.realmofthemadgod.com/version.txt")
	checkErr(err)
	defer resp.Body.Close()

	vers, err := ioutil.ReadAll(resp.Body)
	checkErr(err)

	localVers, err := ioutil.ReadFile(workingPath + "lib/version.txt")
	checkErr(err)

	if string(localVers) == string(vers) {
		fmt.Println("Game not updated, still on build", string(vers))
		checkMenu()
		return
	}

	version = string(vers)
	err = ioutil.WriteFile(workingPath+"lib/version.txt", vers, 0777)
	checkErr(err)

	fmt.Println("Game updated from", string(localVers), "to", string(vers))
	downloadClient(true, true)
}

func downloadClient(update, menu bool) {
	if !update {
		download(workingPath + "lib/version.txt", "http://www.realmofthemadgod.com/version.txt")

		version_, err := ioutil.ReadFile(workingPath + "lib/version.txt")
		checkErr(err)
		version = string(version_)
	}

	download(workingPath + "client" + version + ".swf", "http://www.realmofthemadgod.com/AssembleeGameClient" + version + ".swf")

	fmt.Println("Client [" + version + "] saved.")
	if menu {
		checkMenu()
	}
}

func addProxy() {
	if _, err := os.Stat(workingPath + "client" + version + ".swf"); os.IsNotExist(err) {
		downloadClient(false, false)
	}

	java, err := exec.LookPath("java")
	checkErr(err)

	err = exec.Command(java, "-jar", "ffdec.jar", "-selectclass", "kabam.rotmg.servers.control.ParseServerDataCommand", "-export", "script", workingPath+"decompiled"+version, workingPath+"client"+version+".swf").Run()
	checkErr(err)

	file, err := ioutil.ReadFile(workingPath + "decompiled" + version + "/scripts/kabam/rotmg/servers/control/ParseServerDataCommand.as")
	checkErr(err)
	content := string(file)

	r := regexp.MustCompile("return _loc(\\d+)_;[\\s\\S]*?}")

	content = r.ReplaceAllString(content, "_loc${1}_.push(this.LocalhostServer());\n\t return _loc${1}_\n\t}\n\n\tpublic function LocalhostServer():Server\n\t{\n\treturn new Server().setName(\"## Proxy Server ##\").setAddress(\"127.0.0.1\").setPort(Parameters.PORT).setLatLong(Number(50000),Number(50000)).setUsage(0).setIsAdminOnly(false);\n\t}")

	ioutil.WriteFile(workingPath+"decompiled"+version+"/scripts/kabam/rotmg/servers/control/ParseServerDataCommand.as", []byte(content), 0777)

	err = exec.Command(java, "-jar", "ffdec.jar", "-replace", workingPath+"client"+version+".swf", workingPath+"client"+version+".swf", "kabam.rotmg.servers.control.ParseServerDataCommand", workingPath+"decompiled"+version+"/scripts/kabam/rotmg/servers/control/ParseServerDataCommand.as").Run()
	checkErr(err)

	fmt.Println("Proxy added.")
	checkMenu()
}

func exportResources() {
	fmt.Println("Exporting started")
	if _, err := os.Stat(workingPath + "client" + version + ".swf"); os.IsNotExist(err) {
		downloadClient(false, false)
	}

	java, err := exec.LookPath("java")
	checkErr(err)

	err = exec.Command(java, "-jar", "ffdec.jar", "-export", "image", workingPath+"decompiled"+version+"/images", workingPath+"client"+version+".swf").Run()
	checkErr(err)

	files, err := ioutil.ReadDir(workingPath + "decompiled" + version + "/images")
	checkErr(err)

	r := regexp.MustCompile("([a-z.]+)(\\w+.jpg|\\w+.png)")

	for _, f := range files {
		if strings.Count(f.Name(), ".") > 1 {
			data, err := ioutil.ReadFile(workingPath + "decompiled" + version + "/images/" + f.Name())
			checkErr(err)

			name := r.FindAllStringSubmatch(f.Name(), -1)
			path := strings.Replace(name[0][1], ".", "/", -1)
			os.MkdirAll(workingPath+"decompiled"+version+"/formatted/"+path, 0777)
			ioutil.WriteFile(workingPath+"decompiled"+version+"/formatted/"+path+name[0][2], data, 0777)
		}
	}

	err = exec.Command(java, "-jar", "ffdec.jar", "-export", "binaryData", workingPath+"decompiled"+version+"/binaryData", workingPath+"client"+version+".swf").Run()
	checkErr(err)

	files, err = ioutil.ReadDir(workingPath + "decompiled" + version + "/binaryData")
	checkErr(err)

	r = regexp.MustCompile("([a-z.]+)(\\w+.bin)")

	for _, f := range files {
		if strings.Count(f.Name(), ".") > 1 {
			data, err := ioutil.ReadFile(workingPath + "decompiled" + version + "/binaryData/" + f.Name())
			checkErr(err)

			matches := r.FindAllStringSubmatch(f.Name(), -1)
			path := strings.Replace(matches[0][1], ".", "/", -1)
			name := strings.Replace(matches[0][2], ".bin", ".dat", -1)
			os.MkdirAll(workingPath+"decompiled"+version+"/formatted/"+path, 0777)
			ioutil.WriteFile(workingPath+"decompiled"+version+"/formatted/"+path+name, data, 0777)
		}
	}

	err = exec.Command(java, "-jar", "ffdec.jar", "-selectclass", "kabam.rotmg.messaging.impl.GameServerConnection,kabam.rotmg.assets.EmbeddedData,kabam.rotmg.assets.EmbeddedAssets", "-export", "script", workingPath+"decompiled"+version, workingPath+"client"+version+".swf").Run()
	checkErr(err)

	gsc, err := ioutil.ReadFile(workingPath + "decompiled" + version + "/scripts/kabam/rotmg/messaging/impl/GameServerConnection.as")
	checkErr(err)
	content := string(gsc)

	r = regexp.MustCompile("const ([\\s\\S]*?):int[\\s\\S]*?(\\d+);")
	matches := r.FindAllStringSubmatch(content, -1)

	as, err := os.Create(workingPath + "decompiled" + version + "/Packets.as")
	checkErr(err)
	defer as.Close()
	xml, err := os.Create(workingPath + "decompiled" + version + "/Packets.xml")
	checkErr(err)
	defer xml.Close()

	i := 0
	xml.WriteString("<Packets>\n")
	for i = 0; i < len(matches); i++ {
		as.WriteString("public static const " + matches[i][1] + ":int = " + matches[i][2] + ";\n")
		xml.WriteString("	<Packet>\n		<PacketName>" + strings.Replace(matches[i][1], "_", "", -1) + "</PacketName>\n		<PacketID>" + matches[i][2] + "</PacketID>\n	</Packet>\n")
	}
	xml.WriteString("</Packets>\n")
	xml.WriteString("<Count>" + strconv.Itoa(i) + "</Count>")
	as.WriteString("// " + strconv.Itoa(i) + " Packets")

	data_, err := ioutil.ReadFile(workingPath + "decompiled" + version + "/scripts/kabam/rotmg/assets/EmbeddedData.as")
	checkErr(err)
	data := string(data_)

	r = regexp.MustCompile("(\\w+)+ static const (\\w+):Class = ([_\\w]+);")

	data = r.ReplaceAllString(data, "[Embed(source=\"${3}.dat\", mimeType=\"application/octet-stream\")]\n\t${1} static const ${2}:Class;")
	ioutil.WriteFile(workingPath+"decompiled"+version+"/formatted/kabam/rotmg/assets/EmbeddedData.as", []byte(data), 0777)

	data_, err = ioutil.ReadFile(workingPath + "decompiled" + version + "/scripts/kabam/rotmg/assets/EmbeddedAssets.as")
	checkErr(err)
	data = string(data_)

	files, err = ioutil.ReadDir(workingPath + "decompiled" + version + "/formatted/kabam/rotmg/assets")
	checkErr(err)

	r = regexp.MustCompile("Class = ([_\\w]+);")
	matches = r.FindAllStringSubmatch(data, -1)

	for i := 0; i < len(matches); i++ {
		name := ""
		r2 := regexp.MustCompile(matches[i][1])
		for _, f := range files{
			if r2.MatchString(f.Name()) {
				name = f.Name()
			}
		}
		if name == "" {
			name = strings.Replace(matches[i][1], "EmbeddedAssets_", "", -1)
			name = strings.Replace(name, "Embed_", "", -1)
			download(workingPath + "decompiled" + version + "/formatted/kabam/rotmg/assets/" + matches[i][1] + ".png", "https://static.drips.pw/rotmg/production/current/sheets/" + name + ".png")
			name = matches[i][1] + ".png"
		}
		if strings.Contains(name, ".dat") {
			r3 := regexp.MustCompile("(\\w+) static (\\w+) (\\w+):Class = " + matches[i][1] + ";")
			data = r3.ReplaceAllString(data, "[Embed(source=\"" + name + "\", mimeType=\"application/octet-stream\")]\n\t${1} static ${2} ${3}:Class;")
		} else {
			r3 := regexp.MustCompile("(\\w+) static (\\w+) (\\w+):Class = " + matches[i][1] + ";")
			data = r3.ReplaceAllString(data, "[Embed(source=\"" + name + "\")]\n\t${1} static ${2} ${3}:Class;")
		}
	}

	ioutil.WriteFile(workingPath+"decompiled"+version+"/formatted/kabam/rotmg/assets/EmbeddedAssets.as", []byte(data), 0777)

	checkMenu()
}

func download(path, link string) {
	file, err := os.Create(path)
	checkErr(err)
	defer file.Close()

	resp, err := http.Get(link)
	checkErr(err)

	_, err = io.Copy(file, resp.Body)
	checkErr(err)
}

func getWorkingModel(model int) {
	switch model {
	case 1:
		checkUpdates()
		return
	case 2:
		downloadClient(false, true)
		return
	case 3:
		exportResources()
		return
	case 4:
		addProxy()
		return
	default:
		fmt.Print("Unknown model.")
	}
}

func checkMenu() {
	fmt.Print(menu)
	var menuInt int
	fmt.Scan(&menuInt)
	getWorkingModel(menuInt)
}

func main() {
	fmt.Println("Available", runtime.GOMAXPROCS(runtime.NumCPU()), "processes.")
	path, err := filepath.Abs("./")
	checkErr(err)
	workingPath = path + "/"
	vers, err := ioutil.ReadFile(workingPath + "lib/version.txt")
	checkErr(err)
	version = string(vers)
	checkMenu()
}
