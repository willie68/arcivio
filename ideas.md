Ich gehe jetzt in einige Themen mal etwas tiefer rein:

Modellierung Record: Atttributwerte eines Dokumentes (Record) können immer mehrfach sein. Das Attribut ist aber immer nur einmal vorhanden. Werte können von verschiedene Typen sein. Z.B. wurde zunächst das Attribute Rechnungsnummer als numerisch modelliert. Später wird es, z.B. Bedingt durch eine Änderung in einer ext. Software in Text geändert. Um zu verhindern das nun alle bereits archvierten Dokumente ungültig werden, muss das System bei der Anzeige die Typen der Attributwerte berücksichtigen. D.h. Anzeige wertet den AttributeValueType aus, Neuanlage oder Editieren den AttributeType vom Model.

Eine Beispielhafte JSON Struktur für einen Record sähe z.B. so aus:
resord.json:
{
    "recordid": "damgsts7hb6j4s0krieg",
    "format": 1,
    "doctype": "invoice",
    "properties": {}

    "attributes": [{
            "name": "createdat",
            "group": "system",
            "values": [{
                    "type": "datetime",
                    "value": "2012-04-23T18:25:43.511Z",
                    "properties": {}
                }
            ]
        }, {
            "name": "owner",
            "group": "system",
            "values": [{
                    "type": "text",
                    "value": "damgsts7hb6j4s0krieh",
                    "properties": {
                        "display_value": "w.klaas@gmx.de"
                    }
                }
            ]
        }, {
            "name": "invoice_number",
            "group": "invoice",
            "values": [{
                    "type": "number",
                    "value": 12345,
                    "properties": {}
                }, {
                    "type": "text",
                    "value": "12345a",
                    "properties": {}
                }
            ]
        }, {
            "name": "invoice_file",
            "group": "invoice",
            "values": [{
                    "type": "file",
                    "value": "damgsts7hb6j4s0kriehi",
                    "properties": {
                        "content_type": "application/pdf",
                        "content_length": 35673,
                        "filename": "invoice260927130456_1.pdf",
                        "folder": "rechnungen/2026/09",
                        "page": 1
                    }
                }, {
                    "type": "file",
                    "value": "damgsts7hb6j4s0kriehh",
                    "properties": {
                        "content_type": "application/pdf",
                        "content_length": 35673,
                        "filename": "invoice260927130456_2.pdf",
                        "folder": "rechnungen/2026/09"
                        "page": 2
                    }
                }
            ]
        }, {
            "name": "invoice_ocr",
            "group": "invoice",
            "values": [{
                    "type": "file",
                    "value": "damgsts7hb6j4s0kriehj",
                    "properties": {
                        "content_type": "text/simple",
                        "content_length": 5673,
                        "filename": "fulltext.txt",
                        "referenceto": "damgsts7hb6j4s0kriehi"
                    }
                }, {
                    "type": "file",
                    "value": "damgsts7hb6j4s0kriehk",
                    "properties": {
                        "content_type": "text/simple",
                        "content_length": 3465,
                        "filename": "fulltext.txt",
                        "referenceto": "damgsts7hb6j4s0kriehj"
                    }
                }
            ]
        }
    ]
}