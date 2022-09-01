package main

import (
	"fmt"
	"github.com/unidoc/unipdf/v3/annotator"
	"github.com/unidoc/unipdf/v3/common/license"
	"github.com/unidoc/unipdf/v3/core"
	"github.com/unidoc/unipdf/v3/fjson"
	"github.com/unidoc/unipdf/v3/model"
	"os"
	"strconv"
)

func init() {
	// Make sure to load your metered License API key prior to using the library.
	// If you need a key, you can sign up and create a free one at https://cloud.unidoc.io
	err := license.SetMeteredKey(os.Getenv(`UNIDOC_LICENSE_API_KEY`))
	if err != nil {
		panic(err)
	}
}

func main() {
	if len(os.Args) < 4 {
		fmt.Printf("Syntax: go run pdf_update_fields.go sample_form.pdf sample_form2.pdf formdata.json\n")
	}
	inputPath := os.Args[1]
	outputPath := os.Args[2]
	filejson := os.Args[3]

	err := updateExistingPdfFields(inputPath, outputPath, filejson) //
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

type NewUpdatedData struct {
	Name     string
	Flag     model.FieldFlag
	Font     model.StdFontName
	FontSize int
	Color    string
}

// NewNames represent the new names, Flags to be included, font and font size for the fields of the outputPath from the inputPath.
var NewNames = map[string]NewUpdatedData{
	"name3[first]":         {"firstName", model.FieldFlagMultiline, model.HelveticaBoldObliqueName, 16, "0.000 g"},
	"name3[last]":          {"lastName", model.FieldFlagMultiline, model.TimesItalicName, 12, "1.000 1.000 1.000 rg"},
	"email4":               {"email", model.FieldFlagMultiline, model.TimesBoldItalicName, 28, "0.000 g"},
	"address5[addr_line1]": {"addressL1", model.FieldFlagMultiline, model.CourierName, 10, "0.300 0.400 0.900 rg"},
	"address5[addr_line2]": {"addressL2", model.FieldFlagMultiline, model.CourierBoldObliqueName, 8, "0.300 0.400 0.900 rg"},
	"address5[city]":       {"addressCity", model.FieldFlagMultiline, model.CourierBoldName, 14, "0.300 0.400 0.900 rg"},
	"address5[state]":      {"addressState", model.FieldFlagMultiline, model.HelveticaBoldObliqueName, 16, "0.000 g"},
	"address5[postal]":     {"addressPostal", model.FieldFlagMultiline, model.HelveticaObliqueName, 14, "1.000 1.000 1.000 rg"},
	"fakeSubmitButton":     {"buttonText", model.FieldFlagDoNotScroll, model.ZapfDingbatsName, 12, "0.000 g"},
}

// updateExistingPdfFields The function loads field data from `fileJson` and used to fill in form data in `inputPath` and outputs
// with new field names, flags, font and font size extracted from the NewNames Global Variable.
func updateExistingPdfFields(inputPath, outputPath, fileJson string) error { //
	f, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer f.Close()
	pdfReader, err := model.NewPdfReader(f)
	if err != nil {
		return err
	}
	acroForm := pdfReader.AcroForm
	variableTextDAs := make(map[string]string)
	for _, v := range NewNames {
		newFont := model.NewStandard14FontMustCompile(v.Font)
		acroForm.DR.SetFontByName(*core.MakeName(newFont.BaseFont()), newFont.ToPdfObject())
		variableTextDAs[v.Name] = "/" + newFont.BaseFont() + " " + strconv.Itoa(v.FontSize) + " " + " Tf " + v.Color
	}
	fields := acroForm.AllFields()
	for _, field := range fields {
		if v, ok := NewNames[field.T.String()]; ok {
			name := v.Name
			field.SetFlag(v.Flag)
			objectString := core.MakeString(name)
			field.T = objectString
			l := model.PdfFieldText{
				PdfField: field,
				DA:       core.MakeString(variableTextDAs[name]),
				Q:        field.VariableText.Q,
				DS:       field.VariableText.DS,
				RV:       field.VariableText.RV,
				MaxLen:   nil,
			}
			l2 := model.VariableText{
				DA: core.MakeString(variableTextDAs[name]),
				Q:  field.VariableText.Q,
				DS: field.VariableText.DS,
				RV: field.VariableText.RV,
			}
			field.SetContext(&l)
			field.VariableText = &l2
		}
	}
	// We Extract Fields Data from the fileJson Path.
	fieldsData, err := fjson.LoadFromJSONFile(fileJson)
	if err != nil {
		return err
	}
	fieldFallBacks := make(map[string]*annotator.AppearanceFont)
	fieldAppearance := annotator.FieldAppearance{OnlyIfMissing: false, RegenerateTextFields: true}
	for _, v := range NewNames {
		font, err := model.NewStandard14Font(v.Font)
		if err != nil {
			return err
		}
		fieldFallBacks[v.Name] = &annotator.AppearanceFont{
			Name: font.FontDescriptor().FontName.String(),
			Font: font,
			Size: float64(v.FontSize),
		}
	}

	defaultFontReplacement, err := model.NewStandard14Font(model.TimesItalicName)

	style := fieldAppearance.Style()
	style.Fonts = &annotator.AppearanceFontStyle{
		Fallback: &annotator.AppearanceFont{
			Font: defaultFontReplacement,
			Name: defaultFontReplacement.FontDescriptor().FontName.String(),
			Size: 14,
		},
		FieldFallbacks: fieldFallBacks,
		ForceReplace:   true,
	}
	fieldAppearance.SetStyle(style)

	err = acroForm.FillWithAppearance(fieldsData, fieldAppearance)
	if err != nil {
		return err
	}

	// You can comment to not Flatten if you don't need it.
	//	err = pdfReader.FlattenFields(true, fieldAppearance)
	//	if err != nil {
	//		return err
	//	}

	// The document AcroForm field is no longer needed.
	opt := &model.ReaderToWriterOpts{
		SkipAcroForm: false,
	}

	pdfWriter, err := pdfReader.ToWriter(opt)
	if err != nil {
		return err
	}

	err = pdfWriter.WriteToFile(outputPath)
	if err != nil {
		return err
	}

	return nil
}
