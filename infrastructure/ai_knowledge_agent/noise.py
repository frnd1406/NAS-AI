"""
Noise injection module for OCR-style text corruption.
Applied only to rendered content, NOT to ground truth metadata.
"""

import random
import re
from typing import Optional


# Character substitution mappings (OCR-style errors)
CHAR_SUBSTITUTIONS = {
    'i': ['1', 'l', '|'],
    'l': ['1', 'I', '|'],
    'I': ['1', 'l', '|'],
    'o': ['0', 'O'],
    'O': ['0', 'o'],
    '0': ['O', 'o'],
    'a': ['@', 'ä'],
    'e': ['3', 'ε'],
    's': ['5', '$'],
    'B': ['8', 'ß'],
}

# Common German keywords and their typo variants
KEYWORD_TYPOS = {
    'Rechnung': ['Rechugn', 'Rchnung', 'Rechnugn', 'Rechnnug'],
    'Betrag': ['Betrga', 'Btrag', 'Betrrag'],
    'Gesamt': ['Geasmt', 'Gesammt', 'Gsamt'],
    'Datum': ['Daum', 'Dtaum', 'Datmu'],
    'Kunde': ['Knude', 'Kudne', 'Kude'],
    'Server': ['Sever', 'Serevr', 'Srevr'],
    'Error': ['Erorr', 'Errro', 'Eror'],
    'Warning': ['Warnign', 'Waring', 'Wraning'],
}


def inject_noise(text: str, intensity: float = 0.1, seed: Optional[int] = None) -> str:
    """
    Apply OCR-style noise to text for testing search robustness.
    
    Args:
        text: Original text content
        intensity: Probability of applying noise (0.0 to 1.0)
        seed: Optional random seed for reproducibility
        
    Returns:
        Text with noise injected
    """
    # FIX: Use isolated RNG instance to avoid polluting global random state
    rng = random.Random(seed)
    
    result = text
    
    # Apply keyword typos (word-level)
    for keyword, typos in KEYWORD_TYPOS.items():
        if keyword in result and rng.random() < intensity * 2:
            typo = rng.choice(typos)
            # Replace only first occurrence to keep some readable
            result = result.replace(keyword, typo, 1)
    
    # Apply character substitutions (char-level)
    chars = list(result)
    for i, char in enumerate(chars):
        if char in CHAR_SUBSTITUTIONS and rng.random() < intensity:
            chars[i] = rng.choice(CHAR_SUBSTITUTIONS[char])
    
    result = ''.join(chars)
    
    # Occasional letter swaps
    if rng.random() < intensity:
        words = result.split()
        for j in range(len(words)):
            if len(words[j]) > 3 and rng.random() < intensity:
                # Swap two adjacent letters
                idx = rng.randint(1, len(words[j]) - 2)
                word_chars = list(words[j])
                word_chars[idx], word_chars[idx + 1] = word_chars[idx + 1], word_chars[idx]
                words[j] = ''.join(word_chars)
        result = ' '.join(words)
    
    return result


def add_ocr_artifacts(text: str, intensity: float = 0.05) -> str:
    """
    Add OCR-specific artifacts like random whitespace or missing chars.
    
    Args:
        text: Original text content
        intensity: Probability of adding artifacts
        
    Returns:
        Text with OCR artifacts
    """
    lines = text.split('\n')
    result_lines = []
    
    for line in lines:
        # Occasionally add extra space
        if random.random() < intensity:
            idx = random.randint(0, max(0, len(line) - 1))
            line = line[:idx] + ' ' + line[idx:]
        
        # Occasionally remove a character
        if len(line) > 5 and random.random() < intensity:
            idx = random.randint(0, len(line) - 1)
            line = line[:idx] + line[idx + 1:]
        
        result_lines.append(line)
    
    return '\n'.join(result_lines)
