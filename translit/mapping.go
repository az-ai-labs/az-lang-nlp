package translit

// cyrToLat maps Azerbaijani Cyrillic runes to Latin runes.
// Г/г is excluded — it requires contextual disambiguation (see context.go).
// Ь/ь and Ъ/ъ are handled as skip cases in the conversion loop.
var cyrToLat = map[rune]rune{
	'А': 'A', 'а': 'a',
	'Б': 'B', 'б': 'b',
	'В': 'V', 'в': 'v',
	// Г/г excluded — contextual, see context.go
	'Ғ': 'Ğ', 'ғ': 'ğ',
	'Д': 'D', 'д': 'd',
	'Е': 'E', 'е': 'e',
	'Ә': 'Ə', 'ә': 'ə',
	'Ж': 'J', 'ж': 'j',
	'З': 'Z', 'з': 'z',
	'И': 'İ', 'и': 'i',
	'Й': 'Y', 'й': 'y', // Russian-style short I (compatibility)
	'Ј': 'Y', 'ј': 'y', // Azerbaijani Je (standard)
	'К': 'K', 'к': 'k',
	'Ҝ': 'G', 'ҝ': 'g',
	'Л': 'L', 'л': 'l',
	'М': 'M', 'м': 'm',
	'Н': 'N', 'н': 'n',
	'О': 'O', 'о': 'o',
	'Ө': 'Ö', 'ө': 'ö',
	'П': 'P', 'п': 'p',
	'Р': 'R', 'р': 'r',
	'С': 'S', 'с': 's',
	'Т': 'T', 'т': 't',
	'У': 'U', 'у': 'u',
	'Ү': 'Ü', 'ү': 'ü',
	'Ф': 'F', 'ф': 'f',
	'Х': 'X', 'х': 'x',
	'Һ': 'H', 'һ': 'h',
	'Ч': 'Ç', 'ч': 'ç',
	'Ҹ': 'C', 'ҹ': 'c',
	'Ш': 'Ş', 'ш': 'ş',
	'Ы': 'I', 'ы': 'ı',
}

// latToCyr maps Azerbaijani Latin runes to Cyrillic runes.
var latToCyr = map[rune]rune{
	'A': 'А', 'a': 'а',
	'B': 'Б', 'b': 'б',
	'C': 'Ҹ', 'c': 'ҹ',
	'Ç': 'Ч', 'ç': 'ч',
	'D': 'Д', 'd': 'д',
	'E': 'Е', 'e': 'е',
	'Ə': 'Ә', 'ə': 'ә',
	'F': 'Ф', 'f': 'ф',
	'G': 'Ҝ', 'g': 'ҝ',
	'Ğ': 'Ғ', 'ğ': 'ғ',
	'H': 'Һ', 'h': 'һ',
	'I': 'Ы', 'ı': 'ы', // I/ı (dotless pair)
	'İ': 'И', 'i': 'и', // İ/i (dotted pair)
	'J': 'Ж', 'j': 'ж',
	'K': 'К', 'k': 'к',
	'L': 'Л', 'l': 'л',
	'M': 'М', 'm': 'м',
	'N': 'Н', 'n': 'н',
	'O': 'О', 'o': 'о',
	'Ö': 'Ө', 'ö': 'ө',
	'P': 'П', 'p': 'п',
	'Q': 'Г', 'q': 'г',
	'R': 'Р', 'r': 'р',
	'S': 'С', 's': 'с',
	'Ş': 'Ш', 'ş': 'ш',
	'T': 'Т', 't': 'т',
	'U': 'У', 'u': 'у',
	'Ü': 'Ү', 'ü': 'ү',
	'V': 'В', 'v': 'в',
	'X': 'Х', 'x': 'х',
	'Y': 'Ј', 'y': 'ј',
	'Z': 'З', 'z': 'з',
}

// frontVowels is the set of Cyrillic front vowels for Г/г disambiguation.
var frontVowels = map[rune]bool{
	'Ә': true, 'ә': true,
	'Е': true, 'е': true,
	'И': true, 'и': true,
	'Ө': true, 'ө': true,
	'Ү': true, 'ү': true,
}
