package building

const (
	expectedMinifiedJS   = `const styles=[{Name:"test1",Filename:"test1.css"},{Name:"test2",Filename:"test2.css"}];const defaultStyle="test1.css";const webroot="/chan";const serverTZ=8;const fileTypes=[];`
	expectedUnminifiedJS = `const styles = [{Name: "test1", Filename: "test1.css"},{Name: "test2", Filename: "test2.css"}];
const defaultStyle = "test1.css";
const webroot = "/chan";
const serverTZ = 8;
const fileTypes = [];`
)
